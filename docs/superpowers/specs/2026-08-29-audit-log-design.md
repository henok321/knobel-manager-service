# Audit Log — Design

Date: 2026-08-29
Status: approved, not implemented
Supersedes: `80c8ddd` (feat: per-game audit log with read endpoint, #346), reverted in `656cf3a`

## Problem

Every change to a game — who created, deleted or changed something — must be recorded and readable
by the frontend as a per-game timeline: who changed when which entity.

## Why the previous attempt was reverted

`80c8ddd` implemented a mutation middleware that snapshotted the whole game aggregate before the
handler ran, snapshotted it again afterwards, and diffed the two. It was reverted for size: 2432
lines, of which 503 were a diff engine and its tests.

Its structural costs, which this design must not repeat:

- Two full `Preload("Teams.Players.Scores")` + `Preload("Rounds.Tables.Scores")` loads on every
  mutation, including score writes, the highest-frequency mutation in the app.
- A `responseRecorder` wrapping every response purely to recover the new id from `POST /games`.
- Deletion inferred from "the after-snapshot came back empty".
- Setup detected by string-matching the route pattern.

## Approach

Postgres writes the audit rows. The application's only job is to say who is making the change.

`AFTER INSERT OR UPDATE OR DELETE` row triggers on the five human-facing tables write one
`audit_events` row per changed row. `to_jsonb(OLD)` and `to_jsonb(NEW)` provide before and after for
free, so there is no diff engine and no snapshotting. A GORM plugin sets the actor as a
transaction-local setting before each write; the trigger reads it back.

Rejected alternatives, with reasons, are recorded at the end of this document.

## Verified mechanics

Both load-bearing assumptions were wrong on first contact and were fixed against a real Postgres 18
container running the project's own goose migrations. The throwaway spike proving them is not
checked in; its findings are:

**Actor propagation must anchor before the write, not after the transaction begins.** Registering the
callback at `After("gorm:begin_transaction")` runs it while `Statement.ConnPool` is still the
`*sql.DB` pool. `set_config` then lands on an arbitrary pooled connection and the trigger reads back
`""` — every audit row recorded `system`. Anchoring at `Before("gorm:create")` /
`Before("gorm:update")` / `Before("gorm:delete")` runs the callback with `Statement.ConnPool` as the
`*sql.Tx`, and the value reaches the trigger.

**`pg_trigger_depth()` does not identify cascade deletes.** Referential-integrity cascade deletes fire
their triggers at depth 1, exactly like a direct delete. A `pg_trigger_depth() > 1` guard is dead
code: deleting one game wrote six audit rows, four of them with a `NULL` game_id because their
parents were already gone. Cascades are instead identified by data — a cascaded child can no longer
resolve a live game, a directly deleted child still can.

Also confirmed: the actor does not leak into the next request through the connection pool; a write
inside an explicit `Transaction(...)` (as `GamesRepository.WithinTransaction` uses) carries the actor
correctly; and `game_id` resolves through `teams` for players and through `game_tables → rounds` for
scores.

## Scope

Audited: `games`, `game_owners`, `teams`, `players`, `scores`.

Not audited: `rounds`, `game_tables`, `table_players`. These are derived from the setup algorithm,
not edited by a human. One setup run on a 60-player game writes roughly 380 rows across them, which
would bury the events anyone actually reads.

This section originally claimed that re-running setup wipes scores with no audit trace. That was
wrong, and the error survived long enough to mislead two independent reviewers who read it as
context and confirmed it. `ResetGameTables` deletes `scores` explicitly before dropping
`game_tables`, so each score still resolves its game through parents that are still standing and is
recorded and attributed. Verified by `TestAuditActor`: a setup re-run records one `scores`/`delete`
event per wiped score.

The near miss is instructive. Had that function relied on the cascade instead, the suppression rule
would have erased a tournament's entire scoring history in silence — the one question this log most
needs to answer. The delete order in `ResetGameTables` is load-bearing for the audit log, not just
for referential integrity.

The cascade rule is broader than "a deleted game takes its trail with it", and the wording above
originally hid that. The guard suppresses any delete whose row can no longer reach a live game,
which includes children orphaned by a delete one level up: deleting a team records one `teams`
event, not that event plus one per player it removed. The log is therefore one event per user
action, which is the intended shape — a 60-player game deleted in one request should not produce
hundreds of rows nobody reads — but it means child-level questions ("when did player P disappear?")
are answerable only through the parent event. `TestAuditActor` pins both directions: deleting a team
records exactly one event, and deleting a player directly still records its own.

## Requests that change nothing

`GamesRepository.CreateOrUpdateGame` calls `db.Save`, which emits an `UPDATE` unconditionally —
GORM does not diff against the database — and bumps the GORM-managed `updated_at` that migration
`0004` put on `games`, `teams`, `players` and `scores`. Postgres fires row triggers on that update
even when every column value is identical, because MVCC writes a new tuple regardless.

Left alone, re-submitting an unchanged form would therefore write an audit event whose `old_row` and
`new_row` differ only in `updated_at` — indistinguishable, to a reader, from a real change. Score
entry is the highest-frequency mutation in the app, so this would be the common case rather than the
edge case.

The trigger suppresses it by comparing the rows with `updated_at` removed. `jsonb - text` on a table
without that column is a harmless no-op, so one expression covers all five audited tables. A change
that sets a field back to its previous value in the same statement is correctly suppressed too.

Failed requests are a different matter and remain unrecorded: validation (400), authorization (403)
and missing-row (404) failures never reach a write, so no trigger fires. A repeated attempt to alter
someone else's game leaves no trace. Recording those needs an application-level write, not a trigger,
and is out of scope until someone asks for it.

## Retention

`audit_events.game_id` is a plain nullable integer with no foreign key.

A foreign key with `ON DELETE CASCADE` is impossible here rather than merely undesirable: the
`games` / `delete` row necessarily references a game that no longer exists, so the cascade would
erase the deletion event at the moment it was written. Without the key, deleting a tournament leaves
its trail intact and the deletion itself is recorded.

Rows for deleted games are never pruned. Fine at tournament volume; a retention job is the upgrade
path.

## Schema

`db_migration/0011_audit_events.sql`.

```sql
CREATE TABLE audit_events (
    id bigserial PRIMARY KEY,
    game_id integer,
    table_name text NOT NULL,
    row_id text NOT NULL,
    action text NOT NULL,
    actor_sub text NOT NULL,
    actor_email text NOT NULL,
    old_row jsonb,
    new_row jsonb,
    created_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_game_id_id ON audit_events (game_id, id DESC);
```

No foreign key, by the reasoning above. The index serves the only read query and is already the
right shape for keyset pagination when it is needed.

The trigger function, verified as written:

```sql
CREATE FUNCTION audit_row() RETURNS trigger AS $$
DECLARE
    row_data jsonb;
    resolved_game_id integer;
BEGIN
    IF TG_OP = 'DELETE' THEN
        row_data := to_jsonb(OLD);
    ELSE
        row_data := to_jsonb(NEW);
    END IF;

    resolved_game_id := CASE TG_TABLE_NAME
        WHEN 'games' THEN (row_data->>'id')::integer
        WHEN 'game_owners' THEN (row_data->>'game_id')::integer
        WHEN 'teams' THEN (row_data->>'game_id')::integer
        WHEN 'players' THEN (SELECT game_id FROM teams WHERE id = (row_data->>'team_id')::integer)
        WHEN 'scores' THEN (SELECT r.game_id FROM game_tables gt
                            JOIN rounds r ON r.id = gt.round_id
                            WHERE gt.id = (row_data->>'table_id')::integer)
    END;

    -- Cascade suppression. A child deleted by ON DELETE CASCADE can no longer reach a
    -- live game, because the parent row is already gone; a directly deleted child can.
    -- pg_trigger_depth() cannot make this distinction: RI cascades run at depth 1.
    IF TG_OP = 'DELETE' AND TG_TABLE_NAME <> 'games'
       AND NOT EXISTS (SELECT 1 FROM games WHERE id = resolved_game_id) THEN
        RETURN NULL;
    END IF;

    -- GORM's Save always emits an UPDATE and always bumps updated_at, and Postgres fires
    -- this trigger even when no column value differs. Without this guard, re-saving an
    -- unchanged form writes an event indistinguishable from a real change.
    IF TG_OP = 'UPDATE' AND to_jsonb(OLD) - 'updated_at' = to_jsonb(NEW) - 'updated_at' THEN
        RETURN NULL;
    END IF;

    INSERT INTO audit_events (
        game_id, table_name, row_id, action, actor_sub, actor_email, old_row, new_row
    )
    VALUES (
        resolved_game_id,
        TG_TABLE_NAME,
        COALESCE(row_data->>'id', row_data->>'owner_sub'),
        lower(TG_OP),
        COALESCE(NULLIF(current_setting('app.actor_sub', true), ''), 'system'),
        COALESCE(NULLIF(current_setting('app.actor_email', true), ''), ''),
        CASE WHEN TG_OP = 'INSERT' THEN NULL ELSE to_jsonb(OLD) END,
        CASE WHEN TG_OP = 'DELETE' THEN NULL ELSE to_jsonb(NEW) END
    );

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
```

Plus one `CREATE TRIGGER audit AFTER INSERT OR UPDATE OR DELETE ON <table> FOR EACH ROW EXECUTE
FUNCTION audit_row();` per audited table.

`actor_sub` falls back to `system` rather than failing, so a write made outside a request — a
migration, a manual fix, a future background job — is recorded as having happened rather than
silently skipped.

`row_id` is text because `game_owners` has no surrogate key; `owner_sub` identifies the row.

The `-- +goose StatementBegin` / `StatementEnd` markers are required around the function body, since
its `$$` block contains semicolons.

## Actor propagation

`pkg/audit/actor.go`. `pkg/game` already imports `api/middleware`, so this direction creates no cycle.

```go
type ActorPlugin struct{}

func (ActorPlugin) Name() string { return "audit:actor" }

func (ActorPlugin) Initialize(db *gorm.DB) error {
    setActor := func(tx *gorm.DB) {
        user, ok := middleware.UserFromContext(tx.Statement.Context)
        if !ok {
            return
        }

        _, err := tx.Statement.ConnPool.ExecContext(tx.Statement.Context,
            "SELECT set_config('app.actor_sub', $1, true), set_config('app.actor_email', $2, true)",
            user.Sub, user.Email)
        if err != nil {
            tx.AddError(err)
        }
    }

    if err := db.Callback().Create().Before("gorm:create").Register("audit:actor", setActor); err != nil {
        return err
    }

    if err := db.Callback().Update().Before("gorm:update").Register("audit:actor", setActor); err != nil {
        return err
    }

    return db.Callback().Delete().Before("gorm:delete").Register("audit:actor", setActor)
}
```

Both settings are set in one statement, so this costs one extra round trip per write, not two.

The anchor choice is the whole correctness of this file and is not obvious from reading it, so it
carries a comment naming what breaks at `After("gorm:begin_transaction")`.

Registered in `api/routes/routes.go` alongside the other wiring, because `cmd/main.go` and
`integrationtests` both call `SetupRouter` and neither should be able to forget it. Registering at
each `gorm.Open` site instead would let the integration tests pass while silently recording `system`.

`actor_email` is snapshotted rather than resolved from Firebase on read: it is the historical truth,
it avoids a Firebase lookup per event, and it goes stale if someone changes their address — the right
trade for an audit trail.

## Failure behaviour

A failed audit insert aborts the user's transaction. This is a deliberate reversal of the reverted
design, which logged and swallowed audit failures so that "a broken audit table must not break a
tournament".

The reversal is safe because the failure modes differ. The old middleware could fail on its own
snapshot queries, its own JSON encoding, or a cancelled request context — all independent of whether
the mutation succeeded, and all recoverable. The trigger's insert has no constraints beyond NOT NULLs
it always satisfies, and runs on the connection that just performed the write. The realistic
remaining failure is the disk, which fails the mutation anyway.

Making it best-effort would mean an `EXCEPTION WHEN OTHERS THEN RETURN NULL` block that silently
drops audit rows — worse, because the log would then be quietly incomplete rather than loudly broken.

## Read endpoint

`GET /games/{gameID}/audit`, tag `Audit`, newest first. Add `Audit` to `include-tags` in
`openapi/config/api.yaml`, then `make openapi-generate`.

```yaml
AuditEvent:
  type: object
  properties:
    id: { type: integer, format: int64 }
    entity: { type: string, example: scores }
    entityID: { type: string, example: "42" }
    action: { type: string, enum: [insert, update, delete] }
    actorSub: { type: string }
    actorEmail: { type: string }
    createdAt: { type: string, format: date-time }
    old: { type: object, nullable: true }
    new: { type: object, nullable: true }
  required: [id, entity, entityID, action, actorSub, createdAt]
```

`entity` carries the table name verbatim (`games`, `game_owners`, `teams`, `players`, `scores`). No
singularising map — it would be five lines of translation buying nothing.

`old` and `new` are returned as objects. The server does not compute a field-level diff: the frontend
can compare two objects, and computing it here would reintroduce the diff engine this design exists
to delete. Add it server-side only if more than one consumer needs the same answer.

`entity.AuditEvent` holds the two jsonb columns as `string`, and the converter unmarshals each into a
`map[string]any` for the response. An unreadable row degrades to a null `old`/`new` on that one event
with an error logged, rather than failing the whole log — the columns are written only by the trigger,
so unparseable JSON means a corrupted row, not a normal case.

Layers follow the existing domain module pattern:

- `pkg/audit/repository.go` — `FindByGameID(ctx, gameID) ([]entity.AuditEvent, error)`,
  `Where("game_id = ?", gameID).Order("id DESC")`.
- `pkg/audit/service.go` — `FindByGameID`, delegating.
- `api/handlers/audit_handler.go` — `AuditHandler`, embedded into `apiServer` in `routes.go`.

Authorization is the game ownership check, exactly as the reverted handler did it: call
`gamesService.FindByID(ctx, gameID, sub)` first and let `respondError` map `apperror.ErrNotOwner` to
403 and `apperror.ErrGameNotFound` to 404. No new sentinel errors.

The query is unbounded, matching every other list endpoint here. The `(game_id, id DESC)` index
already supports adding a limit and a keyset cursor when a game outgrows a few hundred events.

## Testing

Integration tests in `integrationtests/audit_test.go`, driven through the real HTTP API against the
real migrations, per the project's preference for integration over unit tests. Table-driven with
`t.Run`. No unit tests: there is no algorithmic complexity left to isolate — the logic lives in SQL
and only exists in a real database.

Behaviours that must be covered, each of which corresponds to something that was actually broken
during the spike:

1. Creating a game records one `games` / `insert` row carrying the caller's real sub, not `system`.
2. Updating a game records `old` and `new` with the before and after values.
3. A player insert resolves `game_id` through `teams`; a score insert resolves it through
   `game_tables → rounds`.
4. Deleting a player directly records one `players` / `delete` row with the correct `game_id`.
5. Deleting a game records exactly one row, not one per cascaded child.
6. A write with no authenticated user records `system` — the actor does not bleed across pooled
   connections from a previous request.
7. Setup writes no `rounds` / `game_tables` / `table_players` rows.
8. `GET /games/{id}/audit` returns newest first; a non-owner gets 403; an unknown game gets 404.

Item 6 is the one a reviewer is most likely to consider paranoid and is the one that caught a real
defect.

## Size

Roughly 70 lines of SQL, 30 lines of Go on the write path, and about 120 lines of read path plus
generated code — against 2432 lines reverted.

## Rejected alternatives

**GORM callbacks building the audit row in Go.** Trades 70 lines of SQL for materially more Go:
`Statement.ReflectValue` handling per operation, and GORM does not expose old values on `Updates`
with a map, which forces a read-before-write and reintroduces the extra query this design removes.
Existing community plugins for this are unmaintained.

**`last_edited_by` column only.** Answers "who last touched this" in eight columns and no new tables,
but not "what happened over time". Undershoots the requirement. It composes with this design later if
a cheap "who last" read is ever wanted without querying the log.

**supa_audit.** A Postgres extension. The service runs stock `postgres:18` from `compose.yaml`,
rendered onto the VPS by `deploy/deploy.yml`. Adopting it means building and publishing a custom
Postgres image and threading it through an ansible deploy in another repository — real infrastructure
risk for what 70 lines of inline SQL does.

**pgMemento.** Needs no extension, but installs its own schema, a transaction log, a row log, an
`audit_id` column on every audited table, and versioning and restore machinery the project would then
own. Several times more code than the 2432 lines just reverted, merely written by someone else.

**Repeating the reverted middleware.** Covered above.

## Implementation order

1. `db_migration/0011_audit_events.sql` — table, function, five triggers.
2. `pkg/audit/actor.go` — the plugin; register it in `routes.SetupRouter`.
3. Integration tests 1–7 (write path). These fail until 1 and 2 are in.
4. `entity.AuditEvent`, `pkg/audit/repository.go`, `pkg/audit/service.go`.
5. `openapi/openapi.yaml` + `openapi/config/api.yaml`, then `make openapi-generate`, review
   `git diff gen/`, commit spec and generated code together.
6. `api/handlers/audit_handler.go` + converter; embed into `apiServer`.
7. Integration test 8 (read path).
