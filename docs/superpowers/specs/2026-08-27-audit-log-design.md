# Audit Log Design

Per-game audit trail recording who created, changed, or deleted something, plus a read endpoint
for the frontend.

## Goal

For any game, answer: **who** changed **which entity**, **when**, and **from what to what**.

## Approach: snapshot-diff middleware

One middleware in the authenticated chain snapshots the full game aggregate before the handler
runs, snapshots it again after, and writes one audit row per changed entity.

```text
PUT /games/3/teams/5
  ├─ before := GamesRepository.FindByID(3)   ← full aggregate, one preload
  ├─ next.ServeHTTP(...)                     ← business logic untouched
  ├─ status 2xx?
  └─ diff(before, FindByID(3))               → [team:5 update name "A"→"B"]
```

Services, handlers, and repositories are not modified. Every current and future mutating endpoint
is audited without a call site.

### Why not the alternatives

**GORM callbacks / hooks.** Rejected. Four blockers, all verifiable in this repo:

1. Every write is `Save(&struct)` (`game/repository.go:54`, `team/repository.go:31`,
   `player/repository.go:31`, `table/repository.go:41`). With a non-zero primary key that is a bare
   `UPDATE ... SET <all fields>` with no preceding `SELECT`, so `Statement.Changed()` has nothing to
   compare against. Getting a "before" means an extra read inside the callback — the same cost as
   this design, paid per row instead of once per request.
2. `0001_init.sql` puts `ON DELETE CASCADE` on every foreign key. `DeleteTeam` is one
   `DELETE FROM teams`; Postgres removes that team's players, their scores, and their table_players.
   GORM fires exactly one callback. Deleting a four-player team would record one row instead of five.
3. `ResetGameTables` (`game/repository.go:95-127`) is four batch deletes of the form
   `Where("table_id IN ?", tableIDs).Delete(&entity.Score{})`. Each callback receives an empty struct
   plus a `WHERE` clause — it never learns which rows went away.
4. Only `teams` and `rounds` carry a `game_id`. A `Score` callback needs
   score → game_tables → rounds → games just to know which game it belongs to, i.e. a join inside
   every write transaction.

GORM's one genuine advantage — callbacks run inside the mutation's transaction, so audit rows commit
atomically — is not enough to outweigh the above. Making it work would mean rewriting persistence to
serve auditing (`Save()` → load-then-`Updates()` everywhere, DB cascades replaced by app-level
deletes), a far larger change that inverts who serves whom.

**Audit calls in the 13 service methods.** Exact attribution and transactional, but 13 call sites
across four services, audit logic braided into business logic, and every new endpoint is a chance to
forget. Kept as the documented upgrade path if the mis-attribution window below ever bites.

### Accepted costs

- **Best-effort, not transactional.** The mutation commits, then the audit write happens. A failed
  audit write is logged and produces a missing entry; it never fails the request. A broken audit
  table must not break a tournament.
- **Mis-attribution window.** Two owners mutating the same game between the two snapshots can be
  credited with each other's change. Upgrade path: service-level auditing.
- **One wasted aggregate read on denied mutations.** The before-snapshot is taken before the handler
  decides on authorization. Nothing leaks, because rows are only written on 2xx.

## Data model

`db_migration/0009_audit_events.sql`, Up-only to match every existing migration:

```sql
-- +goose Up

CREATE TYPE audit_action AS ENUM ('create', 'update', 'delete', 'setup');

CREATE TYPE audit_entity AS ENUM ('game', 'owner', 'team', 'player', 'score');

CREATE TABLE audit_events
(
    id bigserial PRIMARY KEY,
    game_id integer NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    request_id varchar(32) NOT NULL,
    actor_sub varchar(255) NOT NULL,
    actor_email varchar(255) NOT NULL,
    action audit_action NOT NULL,
    entity audit_entity NOT NULL,
    entity_id varchar(255) NOT NULL,
    changes jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_game_id_id ON audit_events (game_id, id DESC);
```

Notes on the column choices:

- `game_id` cascades. Deleting a game erases its audit trail, including the deletion itself. This is
  a deliberate decision — see "Game deletion" below.
- `entity_id` is `varchar(255)`, not `integer`: an owner's identity is a Firebase sub, and 255
  matches `game_owners.owner_sub` so a long UID can never truncate.
- `action` and `entity` are Postgres `ENUM` types, matching `game_status` in `0001_init.sql`. The
  table is append-only and there is no update path in this design, so a bad value would be permanent
  and would render as garbage in the frontend forever — the database is the right place to reject it.
  Adding a value later is `ALTER TYPE audit_action ADD VALUE 'x'`, which runs inside goose's
  per-migration transaction on Postgres 18 (the value is unusable until commit, which does not matter
  for a migration that only adds it). **Removing or renaming a value has no `DROP VALUE`** — it means
  creating a replacement type, altering the column, and dropping the old type. Accepted: both sets
  only ever grow, and the one plausible growth is adding `round` and `table` to `audit_entity` if the
  exclusion below is ever reversed.
- `changes` is `jsonb`, mapped to a Go `string` with `gorm:"type:jsonb"`. pgx sends a Go string as
  text and Postgres casts it to `jsonb` on insert into a `jsonb` column, so **no new dependency**
  (`gorm.io/datatypes` is not needed).
- `request_id` comes from the existing `middleware.RequestFromContext` (`logging.go:34`) and groups
  the several rows one request can emit.

Added to `pkg/entity/model.go`, using string enums per the project's enum pattern:

```go
type AuditAction string

const (
    AuditActionCreate AuditAction = "create"
    AuditActionUpdate AuditAction = "update"
    AuditActionDelete AuditAction = "delete"
    AuditActionSetup  AuditAction = "setup"
)

type AuditEntity string

const (
    AuditEntityGame   AuditEntity = "game"
    AuditEntityOwner  AuditEntity = "owner"
    AuditEntityTeam   AuditEntity = "team"
    AuditEntityPlayer AuditEntity = "player"
    AuditEntityScore  AuditEntity = "score"
)

type AuditEvent struct {
    ID         int64  `gorm:"primaryKey"`
    GameID     int    `gorm:"not null"`
    RequestID  string `gorm:"size:32;not null"`
    ActorSub   string `gorm:"size:255;not null"`
    ActorEmail string `gorm:"size:255;not null"`
    Action     AuditAction `gorm:"size:20;not null"`
    Entity     AuditEntity `gorm:"size:20;not null"`
    EntityID   string      `gorm:"size:255;not null"`
    Changes    string      `gorm:"type:jsonb;not null"`
    CreatedAt  time.Time
}
```

The `size:` tags on `Action` and `Entity` are cosmetic, matching how `Game.Status` is declared for
its `game_status` enum column. goose owns the schema; GORM never automigrates here.

## Scope: what is audited

Diffed entities: **game, owner, team, player, score.**

Deliberately excluded: **round, table, table_players.** Those are output of the table-assignment
algorithm, not human edits. Without the exclusion, one setup click emits roughly 130 rows for a
5-round, 5-table game (25 tables plus about 100 `table_players`).

Instead, `POST /games/{gameID}/setup` emits a single `action=setup, entity=game` row whose `changes`
is an empty array — the action name carries the meaning. Score deletions caused by `ResetGameTables`
are still recorded, because `score` is in the diffed set, so re-running setup mid-tournament leaves a
visible trail of every wiped score. That destructive case falls out of the diff for free.

Flattened fields per entity:

| Entity | Key | Recorded fields |
| --- | --- | --- |
| game | `game:{id}` | `name`, `team_size`, `table_size`, `number_of_rounds`, `status` |
| owner | `owner:{sub}` | presence only (create/delete, empty `changes`) |
| team | `team:{id}` | `name` |
| player | `player:{id}` | `name`, `team_id` |
| score | `score:{id}` | `score`, `player_id`, `table_id` |

## Diff format

```json
[{"field": "name", "from": "Team A", "to": "Die Knobelkoenige"}]
```

- **create** — every field, `from: null`
- **delete** — every field, `to: null`
- **update** — changed fields only

All values are stringified. This keeps the OpenAPI schema concrete (`from` and `to` as nullable
strings) instead of `map[string]interface{}`, which the project's anti-patterns list forbids, and the
frontend renders strings anyway.

`pkg/audit/diff.go` holds two pure functions with no database access:

```go
type recordKey struct {
    Entity entity.AuditEntity
    ID     string
}

type fieldChange struct {
    Field string  `json:"field"`
    From  *string `json:"from"`
    To    *string `json:"to"`
}

type entityChange struct {
    Key     recordKey
    Action  entity.AuditAction
    Changes []fieldChange
}

func flatten(game entity.Game) map[recordKey]map[string]string
func diff(before, after map[recordKey]map[string]string) []entityChange
```

`flatten` is one small explicit function per entity type (~40 lines total); `diff` is generic over
the maps (~30 lines). Keys present only in `after` are creates, only in `before` are deletes, in both
with differing values are updates. `audit.Service.Record` turns each `entityChange` into one
`entity.AuditEvent`, marshalling `Changes` to the `jsonb` column.

## Middleware

Lives in `pkg/audit/middleware.go`, **not** `api/middleware/`. `pkg/game/service.go:11` imports
`api/middleware` for the `FirebaseAuth` interface, so `api/middleware` → `pkg/game` would be an
import cycle. (The root cause is a domain package depending on the HTTP layer; out of scope here.)

```text
skip unless the method mutates (POST, PUT, PATCH, DELETE)
actor  := middleware.UserFromContext(ctx)
reqID  := middleware.RequestFromContext(ctx).ID
gameID := r.PathValue("gameID")
before := svc.Snapshot(ctx, gameID)          // empty when gameID is absent or unknown
next.ServeHTTP(recorder, r)
if recorder.status is not 2xx                → return
if r.Pattern == "DELETE /games/{gameID}"     → return
if r.Pattern == "POST /games"                → created id from response body, one create row
otherwise                                    → diff(before, svc.Snapshot(ctx, gameID))
any error                                    → slog.ErrorContext, response untouched
```

Chain order in `api/routes/routes.go`:

```text
SecurityHeaders → Metrics → RequestLogging → Authentication → Audit
```

Audit must come last: it needs the actor from `Authentication` and the request id from
`RequestLogging`.

`Snapshot` returns an empty `entity.Game` both when the path has no `gameID` (`POST /games`) and
when the id names a game that does not exist — the handler will answer 404 in that case and the 2xx
guard means nothing is written.

Two small mechanical pieces:

- A `ResponseWriter` wrapper capturing the status code (~12 lines; nothing reusable exists in the
  repo — `metrics.go` delegates to `promhttp`).
- Response-body capture, needed **only** for `POST /games`, the single mutating operation without
  `{gameID}` in its path. A two-field struct and one `json.Unmarshal` to recover the created id.

### Game deletion

`DELETE /games/{gameID}` on success writes nothing. The middleware runs after the handler, by which
point the game row is gone and `game_id` would violate the foreign key. This is the accepted
consequence of cascading audit rows with the game.

## Read endpoint

`GET /games/{gameID}/audit` under a new `Audit` tag, which means one line added to
`openapi/config/api.yaml` `include-tags`, a new `api/handlers/audit_handler.go`, and one more
embedded handler in the `apiServer` struct in `api/routes/routes.go`. It is kept out of
`games_handler.go`, already the largest file in the repo at 321 lines.

Schemas added to `openapi/openapi.yaml`:

```yaml
AuditResponse:
  events: [AuditEvent]

AuditEvent:
  id: integer(int64)
  timestamp: string(date-time)
  actor:
    sub: string
    email: string
  action: string, enum [create, update, delete, setup]
  entity: string, enum [game, owner, team, player, score]
  entityId: string
  requestId: string
  changes: [AuditChange]

AuditChange:
  field: string
  from: string, nullable
  to: string, nullable
```

Events are returned newest first (`id DESC`). Rows emitted by one request share a `request_id` and
`created_at`; their relative order within that request is an insertion artefact and carries no
meaning, so the frontend should group by `request_id` rather than read a sequence into it.

Authorization is `gameService.FindByID(ctx, gameID, sub)`, which yields 404 for an unknown game and
403 for a non-owner — the same pattern `TablesHandler` already uses.

## Module layout

`pkg/audit/` follows the project's domain module pattern with concrete exported structs, no
interfaces:

- `repository.go` — `Insert(ctx, []entity.AuditEvent) error`,
  `FindByGameID(ctx, gameID int) ([]entity.AuditEvent, error)` ordered by `id DESC`
- `service.go` — `Snapshot(ctx, gameID)` delegating to `*game.GamesRepository.FindByID`, and
  `Record(ctx, ...)` building and inserting rows
- `diff.go` — pure `flatten` and `diff`
- `middleware.go` — the HTTP middleware

`audit.Service` holds `*game.GamesRepository` and `*AuditRepository`. It does not need
`*game.GamesService`: the ownership check for the read endpoint happens in `AuditHandler`, which
holds `*game.GamesService` the way `TablesHandler` does.

## Deliberate simplifications

Each is marked in the code with a `ponytail:` comment naming its ceiling and upgrade path.

1. **Successful requests only (2xx).** A 403 attempt on someone else's game records nothing. Add an
   `outcome` column when denied attempts need to be visible.
2. **No pagination.** The endpoint returns every event for the game. Add `limit` and a `before`
   cursor once a game exceeds a few hundred events.
3. **`actor_email` snapshotted at write time.** Historical truth, and no Firebase lookup on read.
   Empty string when the token carries no email claim.

## Testing

`pkg/audit/diff_test.go` — table-driven with `t.Run` subtests, no database. The diff is algorithmic,
which is the project's stated exception to preferring integration tests. Cases: create, update, and
delete per entity type; no-op request; owner add and remove; the setup score wipe.

`integrationtests/audit_test.go` — full request flow against a testcontainers Postgres, following the
existing files in that directory:

- create game → rename team → submit scores → `GET /games/{id}/audit` asserts the event sequence,
  actors, and diffs
- a non-owner receives 403; an unknown game receives 404
- a rejected mutation (400) writes no events
- re-running setup records the deleted scores

## Files

| New | Touched |
| --- | --- |
| `db_migration/0009_audit_events.sql` | `pkg/entity/model.go` |
| `pkg/audit/repository.go` | `api/handlers/converters.go` |
| `pkg/audit/service.go` | `api/routes/routes.go` |
| `pkg/audit/diff.go` | `openapi/openapi.yaml` |
| `pkg/audit/middleware.go` | `openapi/config/api.yaml` |
| `pkg/audit/diff_test.go` | `gen/api/api.gen.go` (regenerated) |
| `api/handlers/audit_handler.go` | `CLAUDE.md` (audit module + middleware chain) |
| `integrationtests/audit_test.go` | |
