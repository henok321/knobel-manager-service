# Audit Log Design

Per-game audit trail recording who created, changed, or deleted something, a read endpoint, and an
Audit tab in the game view. Spans two repos: `knobel-manager-service` (everything up to "Module
layout") and `knobel-manager-app` (the "Frontend" section).

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
- **The snapshot loads more than the diff uses.** Reusing `GamesRepository.FindByID` also preloads
  `Rounds.Tables.Players` and `Rounds.Tables.Scores`, which `flatten` discards, twice per mutating
  request. Deliberately not optimised: there is no measured problem, and a dedicated narrower query is
  the upgrade path if live score entry ever shows up in a profile.

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

`from` and `to` are declared `required` *and* `nullable: true` in the OpenAPI schema. Nullable alone
makes oapi-codegen emit `omitempty`, which drops the nulls and gives a create event a structurally
different shape from an update event — the client would then have to treat absent and null as the same
thing. Required-and-nullable keeps the explicit `null` on the wire.

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

`snapshot` distinguishes two failures that look alike and must not be conflated:

- **Game not found** (`gorm.ErrRecordNotFound`) is a legitimate empty snapshot. It happens for
  `POST /games`, which has no `gameID` at all, and for a mutation naming a game that does not exist —
  the handler answers 404 and the 2xx guard means nothing is written either way.
- **Any other query failure** is reported as an error and aborts auditing for that request. Returning
  an empty map instead would make the diff read as though every team, player and score had just been
  deleted, and that fabricated trail would commit happily, because the game row still exists so the
  foreign key holds. No event is far better than an inverted one.

The audit work after the handler runs on `context.WithoutCancel(ctx)` plus a 5s timeout. `net/http`
cancels the request context the moment the client hangs up, and by that point the mutation is already
committed — without this, a dropped connection loses the record of a change that really happened.

The post-handler work is `defer`red, so a handler that commits and then panics still leaves a trail.
The repo installs no `recover` middleware, so without the defer the mutation would be durable with
nothing recording it.

Two small mechanical pieces:

- A `ResponseWriter` wrapper capturing the status code (~12 lines; nothing reusable exists in the
  repo — `metrics.go` delegates to `promhttp`). It implements `Unwrap`, without which it would hide
  `http.Flusher` and `http.Hijacker` from everything further in and break `http.ResponseController`.
- Response-body capture, needed **only** for `POST /games`, the single mutating operation without
  `{gameID}` in its path — keyed off the *absence of a path `gameID`*, not a hardcoded route string.
  A two-field struct and one `json.Unmarshal` to recover the created id. That route gets no special
  event: with no before-snapshot the ordinary diff already emits a create for the game and its
  owner.

### Game deletion

`DELETE /games/{gameID}` on success writes nothing. The middleware runs after the handler, by which
point the game row is gone and `game_id` would violate the foreign key. This is the accepted
consequence of cascading audit rows with the game.

Detected from the data, not the route: a game that still exists always yields at least its own record,
so an empty after-snapshot means the game is gone. Matching the literal pattern `DELETE
/games/{gameID}` would silently break if `BaseURL` were ever set or the path renamed, and the failure
mode is ugly — every deletion would then attempt an insert the foreign key rejects, forever. The setup
event matches on the `/setup` suffix for the same reason.

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

## Frontend (knobel-manager-app)

A fifth tab in the game detail view, rendering the audit log as a table.

### Ordering dependency — read this first

`src/store/openapi-config.cjs` pulls the schema from
`https://raw.githubusercontent.com/henok321/knobel-manager-service/main/openapi/openapi.yaml` — the
service repo's **main branch**. So `pnpm api:gen` cannot produce `useGetAuditLogQuery` until the
backend spec change is merged to `main`. CI's `validate-client` job re-runs codegen against that same
URL and fails on any diff, so the frontend branch stays red until the backend lands. This is a fact
of the setup, not a choice: **backend merges first, frontend second.**

### Cache wiring — no new tag type

`baseApi.ts` declares exactly two tag types, `Game` and `Tables`. The audit log needs neither a third
one nor a single edit to any existing mutation. `getAuditLog` *provides* both of the game's tags:

```ts
getAuditLog: {
  providesTags: (_result, _error, arg) => [
    { type: 'Game', id: arg.gameId },
    { type: 'Tables', id: gameTablesTag(arg.gameId) },
  ],
},
```

Every mutation that can produce an audit event already invalidates `{ Game, id: gameId }` (team,
player, owner, game, setup) or `{ Tables, id: game:gameId }` (scores, setup), so the audit log
refetches on its own. This follows the repo's stated rule that tag granularity tracks the client's
read model — and the audit log's read model is *everything about this game*.

### Tab

`src/pages/games/GameDetail/GameViewContent.tsx`: add `'audit'` to `GAME_TABS`, plus a `Tabs.Tab` and
a `Tabs.Panel`. Two things fall out for free — `isGameTab` is derived from the const array so it
accepts the new value without change, and `getDefaultTab` switches over `GameStatus` rather than
`GameTab`, so its `assertNever` is unaffected. Existing `selected_tab_for_game_*` values in
localStorage stay valid.

### Panel

`src/pages/games/panels/AuditPanel/AuditPanel.tsx`, following `RankingsPanel`:

- Mantine `Table`, newest first, columns **When / Who / Action / Entity / Changes**
- `CenterLoader` while loading, `EmptyStateCard` when the log is empty
- Timestamps via `Intl.DateTimeFormat(i18n.language, { dateStyle: 'short', timeStyle: 'medium' })` —
  native, so no date library is added
- No `useMemo`/`useCallback`/`React.memo`; React Compiler is enabled

`action` and `entity` arrive as string unions from `generatedApi.ts`, so their labels **must not** be
looked up with a template literal — the repo forbids dynamic `t()` keys. A sibling
`auditLabels.ts` maps each union member to a static literal key through a `switch` ending in
`assertNever`, which makes a new backend enum value a compile error until the label is wired up. That
file is pure, so it gets `auditLabels.test.ts` under `node --test`, matching `rankingsMapper.test.ts`.

### Rows are flat, not grouped

One table row per audit event, ordered by `id DESC`. Events from the same request share a timestamp
and consecutive ids, so they already render adjacent. Grouping by `requestId` into a single visual
entry is deliberately skipped; add it when a real request routinely emits enough rows to be confusing.

### i18n

New keys in **both** `en/` and `de/` `gameDetail.json`: `tabs.audit`, and an `audit.*` block for the
column headers, the action labels, the entity labels, the empty state, and the `field: from → to`
change line. EN is the source of truth for type augmentation; DE must be filled in the same commit or
`pnpm check` fails on `i18next-cli status`.

### Gate

`pnpm check` (`tsc --noEmit`, `biome ci`, and the three `i18next-cli` passes) plus `pnpm knip` and
`pnpm test`. Note the repo's own git rules differ from the service repo: never commit without asking,
never push without explicit permission, and **no Claude co-author trailer**.

### A path gameID that was decorative

Review turned up a pre-existing defect that this feature depends on: the player routes ignored the
path `gameID` entirely (`players_handler.go` literally declared it as `_ /* gameID */`, and
`PlayersService` authorized against the player's *real* game). So
`PUT /games/999/teams/1/players/1` returned 200, renamed the player — and wrote **no audit event at
all**, because both snapshots were of the nonexistent game 999.

An audit log that is bypassed by editing the URL is not an audit log, so `PlayersService` now takes
the `gameID` and checks that the team or player actually belongs to it, reporting
`apperror.ErrPlayerNotFound` / `ErrTeamNotFound` on a mismatch rather than `ErrNotOwner` — a
mismatched id must not reveal that the entity exists somewhere else.

## Deliberate simplifications

Each is marked in the code with a `ponytail:` comment naming its ceiling and upgrade path.

1. **Successful requests only (2xx).** A 403 attempt on someone else's game records nothing. Add an
   `outcome` column when denied attempts need to be visible.
2. **No pagination.** The endpoint returns every event for the game. Add `limit` and a `before`
   cursor once a game exceeds a few hundred events.
3. **`actor_email` snapshotted at write time.** Historical truth, and no Firebase lookup on read.
   Empty string when the token carries no email claim.
4. **`request_id` is `varchar(64)`, not `varchar(32)`.** The current id is 16 random bytes hex-encoded,
   exactly 32 characters. Since audit writes are log-only, a zero-headroom column means any future
   change to the id format would stop auditing silently rather than loudly.

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

And in `knobel-manager-app`, after the backend is on `main`:

| New | Touched |
| --- | --- |
| `src/pages/games/panels/AuditPanel/AuditPanel.tsx` | `src/pages/games/GameDetail/GameViewContent.tsx` |
| `src/pages/games/panels/AuditPanel/auditLabels.ts` | `src/store/api.ts` (providesTags + hook export) |
| `src/pages/games/panels/AuditPanel/auditLabels.test.ts` | `src/store/generatedApi.ts` (`pnpm api:gen`) |
| | `src/i18n/locales/{en,de}/gameDetail.json` |
