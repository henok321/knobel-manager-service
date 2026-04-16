# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go REST API service for managing "Knobeln/Schocken" (dice game) tournaments. It provides endpoints for
managing games, teams, players, rounds, tables, and scores. The project uses OpenAPI-first design with generated server
code, PostgreSQL for persistence, and Firebase JWT for authentication.

Frontend: [knobel-manager-app](https://github.com/henok321/knobel-manager-app) (React)

## Prerequisites

- [Go](https://go.dev/doc/install)
- [Docker](https://docs.docker.com/get-docker/)
- [pre-commit](https://pre-commit.com/) (`pip install pre-commit`)

## Development Commands

### Initial Setup

```bash
make setup
```

Sets up the development environment: installs pre-commit hooks, starts PostgreSQL via Docker Compose, and creates `.env`
file with credentials. Database migrations run automatically at application startup via `goose`.

Before running setup, download Firebase credentials from the Firebase Console and save as `firebaseServiceAccount.json`
in the project root.

### Database

```bash
make reset                    # Stops and removes docker database, uninstalls pre-commit hooks
docker compose up -d          # Start PostgreSQL manually
docker compose down --volumes # Stop and remove database
```

Migrations are in `db_migration/` and use `goose`. They run automatically when the application starts (`cmd/main.go`),
not during `make setup`.

### OpenAPI Code Generation

```bash
make openapi-generate   # Generate code from spec (when spec changes)
make openapi-validate   # Validate generated code matches spec (CI/CD)
```

Generated code is **checked into git** in the `gen/` directory. This approach:

- Makes code reviews easier (see exactly what changed)
- Speeds up CI/CD (validation instead of generation)
- Tracks generated code changes in git history

**Workflow:**

1. Edit `openapi/openapi.yaml`
2. Run `make openapi-generate`
3. Review changes with `git diff gen/`
4. Commit both spec and generated code together

**Validation:**

CI/CD runs `make openapi-validate` in parallel with lint and test jobs. The build job only proceeds if all three pass.

### Linting

```bash
make lint       # Runs go fmt and golangci-lint (Go only, fast)
make lint-all   # Runs all pre-commit hooks (golangci-lint, sqlfluff, shellcheck, markdownlint, etc.)
```

`make lint` is for quick Go-only linting during development. `make lint-all` runs the complete pre-commit hook suite for
comprehensive validation before committing.

### Testing

```bash
make test                        # Runs all tests
go test -v ./...                # Run tests directly (same as make test)
go test -v ./pkg/game/...       # Run specific package tests
go test -v -run TestName ./...  # Run specific test
go test -race ./...             # Run tests with race detector

# Coverage (manual)
go test ./... -coverpkg=./... -coverprofile=coverage.out  # Generate coverage
go tool cover -html=coverage.out                           # View HTML report
go tool cover -func=coverage.out                           # View text report
```

### Building

```bash
make build     # Builds binary
make clean     # Removes build artifacts
```

### Running the Application

```bash
# Load environment variables
set -o allexport
source .env
set +o allexport

# Run from source
go run cmd/main.go

# Or run built binary
./knobel-manager-service
```

The service starts two servers:

- Main API server: `http://localhost:8080`
- Metrics server: `http://localhost:9090/metrics` (Prometheus)

Health check: `curl http://localhost:8080/health`

The app also serves the OpenAPI spec at `/openapi.yaml` and Swagger UI at `/docs`.

### Update Dependencies

```bash
make update  # Updates Go modules (go get -u && go mod tidy)
```

### Available Commands

```bash
make help    # Display all available Makefile targets
```

## Architecture

### System Overview

```mermaid
graph TB
    Client[Client App<br/>React Frontend]
    API[Knobel Manager Service<br/>Go REST Service<br/>:8080]
    DB[(PostgreSQL<br/>Database)]
    Firebase[Firebase Auth<br/>JWT Validation]
    Metrics[Prometheus<br/>Metrics<br/>:9090]
    Client -->|HTTP + JWT Bearer Token| API
    Client -.->|Authenticate| Firebase
    API -->|Validate JWT| Firebase
    API -->|SQL Queries| DB
    API -->|Export Metrics| Metrics
```

The system uses:

- **OpenAPI-First**: Server interfaces generated from `openapi/openapi.yaml` using `oapi-codegen`
- **Database**: PostgreSQL with GORM, migrations via `goose`
- **Authentication**: Firebase JWT tokens validated on each request
- **Deployment**: GitHub Actions CI/CD pipeline
- **Monitoring**: Prometheus metrics at `:9090/metrics`, health endpoints at `:8080/health/live` (liveness) and
  `:8080/health/ready` (readiness)

### Code Organization

```shell
cmd/                    # Application entry point
  main.go              # Server initialization, Firebase setup, DB migrations, routing

api/                   # HTTP layer
  routes/              # HTTP route setup with middleware chain
  handlers/            # HTTP handlers implementing OpenAPI interfaces
  middleware/          # Authentication, logging, metrics, rate limiting, security headers
  health/              # Health check implementations (DB, Firebase)
  logging/             # Structured logging with context

pkg/                   # Domain modules (independent, reusable)
  game/                # Game management
  team/                # Team management
  player/              # Player management
  table/               # Table/round management (also handles scores)
  setup/               # Game setup algorithms (table assignments)
  entity/              # Shared database models
  apperror/            # Application sentinel errors

gen/                   # OpenAPI-generated code (DO NOT EDIT MANUALLY)
  health/, games/, teams/, players/, tables/, scores/
                       # Generated types, handler interfaces and routing

openapi/               # OpenAPI specification
  openapi.yaml         # Main OpenAPI spec
  swagger.html         # Swagger UI served at /docs
  config/              # oapi-codegen configuration files per module

db_migration/          # Database migrations (goose)
integrationtests/      # Integration tests using testcontainers
```

### Domain Module Pattern

Each domain module (`pkg/game`, `pkg/team`, `pkg/player`, `pkg/table`) follows this structure:

- `init.go` - Module initialization function (wires repository → service)
- `repository.go` - Database operations (GORM)
- `service.go` - Business logic

Modules are initialized in `api/routes/routes.go` and injected into handlers. Note: scores are handled by
`TablesHandler` — there is no separate scores domain module in `pkg/`.

### OpenAPI-First Development

1. Edit `openapi/openapi.yaml` to add/modify endpoints
2. Update relevant config files in `openapi/config/` if needed
3. Run `make openapi-generate` to regenerate server interfaces
4. Review generated code changes with `git diff gen/`
5. Implement new interfaces in `api/handlers/`
6. Wire up routes in `api/routes/routes.go`
7. Commit both spec and generated code together

The generated code in `gen/` provides:

- Type-safe request/response models
- Server interfaces to implement
- Request validation
- HTTP routing helpers

### Generated Type Usage Pattern

**IMPORTANT: Always use module-specific types.**

oapi-codegen generates types within each module package (`gen/games`, `gen/teams`, `gen/players`, `gen/tables`,
`gen/scores`). Each module contains its own types, ServerInterface definitions, and HTTP routing code. The codebase uses
module-specific types throughout:

**✅ Correct Usage:**

- Use module-specific types: `games.Game`, `teams.Team`, `players.Player`, `tables.Table`, `scores.Score`
- Each module package contains both types AND ServerInterface
- Converters in `api/handlers/converters.go` return module-specific types
- Handlers use module-specific types for requests and responses
- Services use module-specific types for request parameters

**Examples:**

```go
// ✅ Correct - import only the module package you need
import "github.com/henok321/knobel-manager-service/gen/games"

// ✅ Correct - converter returns games.Game
func entityGameToAPIGame(e entity.Game) games.Game { ... }

// ✅ Correct - handler implements games.ServerInterface and uses games.Game
var _ games.ServerInterface = (*GamesHandler)(nil)

func (h *GamesHandler) GetGames(w http.ResponseWriter, r *http.Request) {
apiGames := make([]games.Game, len(allGames))
response := games.GamesResponse{Games: apiGames}
}

// ✅ Correct - service uses games.GameCreateRequest
func (s *gamesService) CreateGame(ctx context.Context, sub string, game *games.GameCreateRequest) (entity.Game, error) { ... }
```

**Why this pattern?**

- Self-contained modules with no cross-dependencies
- Types are scoped to their API context (e.g., `games.Player` represents a player in game responses, while
  `players.Player` represents a player in CRUD operations)
- Simpler configuration - no import-mapping complexity
- Generated code duplication is intentional and correct (same schema, different contexts)

**File references:**

- Module types and interfaces: `gen/games/games.gen.go`, `gen/teams/teams.gen.go`, `gen/players/players.gen.go`,
  `gen/tables/tables.gen.go`, `gen/scores/scores.gen.go`
- Converters: `api/handlers/converters.go`
- Handlers: `api/handlers/*_handler.go`
- Services: `pkg/*/service.go`

### Database Models

Core entities in `pkg/entity/model.go`:

- `Game` - Tournament container with configuration (team size, table size, rounds)
- `GameOwner` - Links games to Firebase user IDs
- `Team` - Group of players
- `Player` - Individual participant
- `Round` - Game round container
- `GameTable` - Table assignment for a round (DB name: `game_tables`)
- `Score` - Player score at a specific table
- `TablePlayer` - Many-to-many join table (DB name: `table_players`)

### Enum Pattern

**Prefer String Enums for readability.**

Unless you are writing mission-critical financial code where a state mismatch costs millions, just use String Enums. The readability benefit outweighs the lack of strict type safety in most Go projects.

String enums are simple, JSON-compatible, database-friendly, and easy to debug. They make code more maintainable by being self-documenting.

### Authentication & Authorization

- Uses Firebase JWT tokens via `Authorization: Bearer <token>` header
- Authentication middleware in `api/middleware/auth.go`
- Extracts user ID (`sub` = Firebase UID) and email from token, stores in request context via `middleware.UserFromContext`
- Authorization checks happen in services (e.g., verifying game ownership via `entity.IsOwner`)
- Application errors use sentinel errors in `pkg/apperror` (e.g., `apperror.ErrNotOwner`, `apperror.ErrTeamNotFound`)

### Middleware Chain

Configured in `api/routes/routes.go`:

- Public endpoints: `RateLimit → SecurityHeaders → Metrics → RequestLogging`
- Authenticated endpoints: `RateLimit → SecurityHeaders → Metrics → RequestLogging → Authentication`

### Scores Architecture Note

The `gen/scores` package is generated from the OpenAPI spec, but **scores are implemented by `TablesHandler`**
(`api/handlers/tables_handler.go`), not a dedicated scores handler. There is no `pkg/scores` domain module.
In `api/routes/routes.go`, `scores.HandlerWithOptions` is wired to `tablesHandler`.

## Test Setup

Integration tests (`integrationtests/`) use:

- `testcontainers-go` to spin up PostgreSQL containers
- Mock Firebase auth client (`integrationtests/mock/auth_mock.go`)
- Real database operations with full goose migrations
- `httptest.Server` wrapping the real router

Tests are automatically run by pre-commit hooks on push and by CI/CD.

## Tools Required

The project uses Go toolchain directives:

- `github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen` - OpenAPI code generation
- `github.com/pressly/goose/v3/cmd/goose` - Database migrations

These are listed in `go.mod` and installed via `go tool <command>`.

## Environment Variables

Required in `.env`:

- `ENVIRONMENT` - Set to `local` for development (enables debug logging)
- `FIREBASE_SECRET` - Base64-encoded Firebase service account JSON
- `DATABASE_URL` - PostgreSQL connection string
- `DB_MIGRATION_DIR` - Path to migrations directory (e.g., `db_migration`)

Optional (with defaults):

- `RATE_LIMIT_REQUESTS_PER_SECOND` - Default: `20`
- `RATE_LIMIT_BURST_SIZE` - Default: `40`
- `RATE_LIMIT_CACHE_DEFAULT_DURATION` - Default: `5m`
- `RATE_LIMIT_CACHE_CLEANUP_PERIOD` - Default: `1m`
- `MAX_REQUEST_SIZE` - Default: `1048576` (1MB in bytes)
- `DB_MAX_OPEN_CONNS` - Default: `25`
- `DB_MAX_IDLE_CONNS` - Default: `5`
- `DB_CONN_MAX_LIFETIME` - Default: `5m`
- `DB_CONN_MAX_IDLE_TIME` - Default: `10m`

## CI/CD

GitHub Actions workflows in `.github/workflows/`:

- **pipeline.yml** - Complete CI/CD pipeline (lint → test → build → deploy)
- **codeql.yml** - Security and code quality analysis (CodeQL)

### Deployment Pipeline

Single workflow runs on push to main with dependent jobs:

1. **Validate, Lint & Test** - Run in parallel
    - Validate OpenAPI: Ensures generated code matches spec (`make openapi-validate`)
    - Lint: Pre-commit hooks (golangci-lint, gitleaks, shellcheck, markdownlint, etc.)
    - Test: Full test suite (`make test`)
2. **Build** - Triggers after all validations pass:
    - Builds multi-arch Docker image (amd64/arm64)
    - Pushes to GitHub Container Registry (`ghcr.io`)
3. **Deploy** - Triggers after successful build:
    - Triggers Coolify deployment via webhook
    - Tracked via GitHub Environments (production)

**On Pull Requests:** Only validation, lint, and test jobs run (build/deploy are skipped)

### Security and Quality Analysis (CodeQL)

- Runs on push to main, PRs, and weekly (Thursday 01:44 UTC)
- Analyzes Go code and GitHub Actions workflows
- Excludes generated code (`gen/`), vendor, tests, migrations
- Results: [GitHub Security](https://github.com/henok321/knobel-manager-service/security)
- `govulncheck` runs in pre-commit hooks for dependency CVE scanning

### Test Coverage

**Local development:**

- Generate coverage: `go test ./... -coverpkg=./... -coverprofile=coverage.out`
- View report: `go tool cover -html=coverage.out`

**Coverage exclusions:** `integrationtests/`, `gen/`, `cmd/`

### Health Verification

```bash
curl https://knobel-manager.com/health
curl https://knobel-manager.com/metrics
```

### Required GitHub Secrets & Variables

- **Secret:** `COOLIFY_API_TOKEN`
- **Variable:** `COOLIFY_DEPLOYMENT_URL`

---

## Code Review Standards

### Review Philosophy

- Be direct and honest in feedback - focus on quality and fact
- Identify security vulnerabilities and bugs as the highest priority
- Focus on code quality and correctness over style preferences
- Suggest improvements that enhance maintainability, but avoid major refactoring unless it significantly improves
  quality
- Follow the boyscout rule: "Leave the campground cleaner than you found it"
- Acknowledge well-implemented patterns and good practices

When reviewing code changes, apply these standards with appropriate severity:

### Critical Issues (Block merge)

- Security vulnerabilities (SQL injection, command injection, exposed secrets)
- Data corruption risks
- Unhandled errors that could crash the service
- Breaking API changes without version bump
- Race conditions or deadlocks

### High Priority Issues (Should fix before merge)

- Incorrect business logic
- Missing authorization checks (verify game ownership in services)
- Inefficient database queries (N+1 queries, missing preloading)
- Missing tests for critical paths
- Violation of project architecture patterns (business logic in handlers, bypassing OpenAPI workflow)

### Medium Priority Issues (Fix or document decision)

- Code duplication
- Missing error context (use fmt.Errorf with %w)
- Unclear variable names
- Suboptimal performance
- Missing edge case handling

### Common Anti-Patterns to Avoid

- Business logic in HTTP handlers (should be in services)
- Direct database access from handlers (use repositories)
- Skipping authorization checks in service layer
- Editing generated code in `gen/` directory
- Using `interface{}` when specific types could be used
- Ignoring errors with `_`
- Not closing resources (missing `defer` for files/connections)
- Hardcoding configuration values
- Creating new sentinel errors instead of using `pkg/apperror`

### Project Best Practices to Encourage

- Following domain module pattern (init.go, repository.go, service.go)
- Clear separation: handler → service → repository
- Using sentinel errors from `pkg/apperror` (e.g., `apperror.ErrNotOwner`, `apperror.ErrTeamNotFound`)
- Comprehensive error handling with context
- Proper transaction handling for multi-step database operations
- Using middleware for cross-cutting concerns
- Structured logging with request context
- Integration tests that verify full request flow
- Prefer integration tests over unit tests unless testing algorithmic complexity
- Tests should test behavior, not implementation details
- Pass context through function calls
- Table-driven tests with t.Run for subtests

---

**Note for Claude Code:** Be direct and honest, do not sugar coat answers, focus on quality and fact.
