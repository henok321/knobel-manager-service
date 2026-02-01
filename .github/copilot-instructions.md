# PR Review Checklist

You are a sensor backend and golang developer. You are responsible for reviewing pull requests and ensuring that they
meet the standards outlined in this document. You will be expected to provide constructive feedback on all pull
requests. You
should focus on the code quality and correctness of the changes, not on style preferences. The goal of the review is to
ensure that the code is maintainable and easy to understand for future contributors. Follow the boyscout rule: "Leave
the campground cleaner than you found it.", but do not suggest major refactoring unless it improves the code quality or
correctness.

This document defines the code review standards for the knobel-manager-service project. It combines general best
practices, Go-specific standards, and project-specific patterns.

## Review Philosophy

- Be direct and honest in feedback
- Focus on code quality and correctness over style preferences
- Identify security vulnerabilities and bugs as the highest priority
- Suggest improvements that enhance maintainability
- Acknowledge well-implemented patterns and good practices

## 1. General Best Practices

### Code Quality

- [ ] Code is self-documenting with clear variable and function names
- [ ] Complex logic has explanatory comments
- [ ] No commented-out code or debug statements left in
- [ ] No hardcoded credentials, secrets, or sensitive data
- [ ] No unnecessary code duplication - common patterns extracted
- [ ] Functions are focused and do one thing well (Single Responsibility)
- [ ] Changes are minimal and focused on the stated purpose

### Error Handling

- [ ] All errors are handled, never silently ignored
- [ ] Error messages provide context about what failed and why
- [ ] Errors are wrapped with additional context using fmt.Errorf
- [ ] Critical errors are logged with appropriate severity levels
- [ ] No panics except for truly unrecoverable situations

### Testing

- [ ] New functionality has corresponding tests
- [ ] Tests cover both happy paths and error cases
- [ ] Test names clearly describe what they test
- [ ] Tests are independent and can run in any order
- [ ] Integration tests verify database operations
- [ ] Tests use meaningful assertions, not just "no error"
- [ ] Prefer integration tests over unit tests unless testing algorithmic complexity
- [ ] Tests should test behavior, not implementation details

### Security

- [ ] No SQL injection vulnerabilities (GORM protects but verifies raw SQL)
- [ ] No command injection risks
- [ ] Input validation on all user-provided data
- [ ] Authorization checks present for protected resources
- [ ] Sensitive data properly sanitized in logs and error messages
- [ ] No exposure of internal implementation details in API responses

### Documentation

- [ ] README updated if user-facing changes
- [ ] OpenAPI spec updated for API changes
- [ ] Code comments explain "why" not just "what"
- [ ] Complex algorithms or business logic documented

## 2. Go-Specific Standards

### Idiomatic Go

- [ ] Follows Go naming conventions (camelCase for unexported, PascalCase for exported)
- [ ] Short variable names in limited scopes (i, err, db)
- [ ] Descriptive names for package-level or long-lived variables
- [ ] Interface names end in -er when appropriate (Reader, Writer, Handler)
- [ ] Receiver names are consistent (typically 1–2 letters)

### Error Handling (Go-specific)

- [ ] Uses Go 1.13+ error wrapping (fmt.Errorf with %w)
- [ ] Error types implement the error interface properly
- [ ] Sentinel errors are package-level variables
- [ ] Error checking happens immediately after function calls
- [ ] No naked returns in functions with named return values and error handling

### Concurrency

- [ ] Goroutines properly handle context cancellation
- [ ] Channels are closed by the sender, not the receiver
- [ ] WaitGroups used correctly for synchronization
- [ ] No goroutine leaks - all started goroutines eventually exit
- [ ] Shared state protected by mutexes or channels
- [ ] Race conditions avoided (run with-race flag)

### Resource Management

- [ ] Files, connections, and HTTP response bodies closed with `defer`
- [ ] Defer statements placed immediately after resource acquisition
- [ ] Database transactions committed or rolled back properly
- [ ] Context passed to functions that need cancellation/timeout

### Performance

- [ ] No unnecessary allocations in hot paths
- [ ] Appropriate data structures (map for lookups, slice for iteration)
- [ ] String concatenation uses strings.Builder for many operations
- [ ] JSON encoding/decoding errors handled
- [ ] Database queries optimized (no N+1 queries, use preloading)
- [ ] Complexity should be only used as a last resort if the performance boost is significant and justified

### Standard Library Usage

- [ ] Uses a standard library where possible over third-party packages
- [ ] Context propagated through a call chain for request-scoped values
- [ ] HTTP status codes from net/http constants
- [ ] Time operations use time.Time and time.Duration appropriately

### Code Organization

- [ ] Package names are lowercase, single word
- [ ] No cyclic dependencies between packages
- [ ] Internal packages used for non-exported code
- [ ] Interfaces defined in consuming package, not implementing package
- [ ] Init functions are used sparingly and only when necessary

## 3. Project-Specific Standards

### OpenAPI-First Development

- [ ] API changes start with openapi/openapi.yaml updates
- [ ] `make openapi-generate` run after spec changes to regenerate code
- [ ] Generated code in gen/ directory checked into git (never edited manually)
- [ ] Review generated code changes with `git diff gen/` before committing
- [ ] Commit both spec and generated code together
- [ ] New endpoints implement generated interfaces from gen/
- [ ] Request/response types match OpenAPI spec exactly
- [ ] Routes wired in internal/routes/routes.go
- [ ] CI validates generated code matches spec with `make openapi-validate`

### Domain Module Pattern

- [ ] Business logic in pkg/{domain}/service.go, not in handlers
- [ ] Database operations in pkg/{domain}/repository.go
- [ ] DTOs defined in pkg/{domain}/model.go
- [ ] Module initialization in pkg/{domain}/init.go
- [ ] Handlers only handle HTTP concerns (parsing, response writing)
- [ ] Services called with context and return errors

### Authentication & Authorization

- [ ] Protected endpoints use Authentication middleware
- [ ] User ID extracted from context using middleware.GetUserID()
- [ ] Authorization verified in service layer, not handlers
- [ ] Resource ownership checked (GameOwner table)
- [ ] UnauthorizedError returned for auth failures
- [ ] Public endpoints are explicitly documented as such

### Error Handling (Project-specific)

- [ ] Uses pkg/apperror types (NotFoundError, ValidationError, etc.)
- [ ] Errors converted to HTTP status via apperror.ToHTTPStatus()
- [ ] Error responses follow a consistent format
- [ ] Database errors properly categorized
- [ ] Client errors (4xx) vs server errors (5xx) distinguished

### Database Operations (GORM)

- [ ] Transactions used for multistep operations
- [ ] Preloading used for associations to avoid N+1 queries
- [ ] Foreign key constraints validated
- [ ] Database migrations in db_migration/ directory
- [ ] Migrations are idempotent and reversible
- [ ] GORM errors checked and handled

### Middleware Chain

- [ ] Public endpoints: Metrics → RequestLogging
- [ ] Authenticated endpoints: Metrics → RequestLogging → Authentication
- [ ] Middleware order preserved as configured in routes.go
- [ ] Context values set by middleware properly extracted

### Logging

- [ ] Structured logging with contextual fields
- [ ] Log levels appropriate (Debug, Info, Warn, Error)
- [ ] Request ID propagated through log context
- [ ] Sensitive data are not logged (passwords, tokens)
- [ ] Errors logged with stack traces when helpful

### Testing (Project-specific)

- [ ] Integration tests in `integrationtests/` directory
- [ ] Uses testcontainers for PostgreSQL
- [ ] Mock Firebase auth via `integrationtests/mock/auth_mock.go`
- [ ] Tests verify database state after operations
- [ ] Tests clean up after themselves (no test pollution)
- [ ] Table-driven tests with t.Run for subtests

### Code Style

- [ ] Code passes `make lint` (pre-commit hooks)
- [ ] Code passes `make lint-go` (golangci-lint)
- [ ] No linter warnings or errors
- [ ] `gofmt` and `goimports` applied
- [ ] Code formatted consistently

### Dependencies

- [ ] New dependencies justified and minimal
- [ ] Dependencies pinned to specific versions in go.mod
- [ ] `go mod tidy` run to clean up unused dependencies
- [ ] Only use stable, well-maintained packages

### Configuration

- [ ] Environment variables documented in CLAUDE.md if new
- [ ] Configuration loaded from .env file
- [ ] No breaking changes to existing environment variables
- [ ] Sensible defaults for non-required config

## 4. Review Process

### Pre-Review Checks (Automated)

- [ ] All tests pass (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Code builds successfully (`make build`)
- [ ] No merge conflicts with target branch
- [ ] Branch is up to date with main

### Code Review Focus Areas

#### Critical Issues (Block merge)

- Security vulnerabilities
- Data corruption risks
- Unhandled errors that could crash the service
- Breaking API changes without a version bump
- Race conditions or deadlocks

#### High Priority Issues (Should fix before merge)

- Incorrect business logic
- Missing authorization checks
- Inefficient database queries
- Missing tests for critical paths
- Violation of project architecture patterns

#### Medium Priority Issues (Fix or document decision)

- Code duplication
- Unclear variable names
- Missing error context
- Suboptimal performance
- Missing edge case handling

#### Low Priority Issues (Nice to have)

- Code style preferences
- Additional test coverage
- Documentation improvements
- Refactoring opportunities

### Merge Criteria

A PR should only be merged when:

- [ ] All critical and high-priority issues resolved
- [ ] All automated checks pass (CI/CD pipeline)
- [ ] Code follows project architecture and patterns
- [ ] Tests provide adequate coverage
- [ ] Documentation updated as needed

## 5. Common Anti-Patterns to Avoid

### Go Anti-Patterns

- Empty interface{} when specific types could be used
- Ignoring errors with _ (except in very specific cases)
- Using init() for complex initialization
- Global mutable state
- Using panic for normal error handling
- Not closing resources (file handles, connections)

### Project Anti-Patterns

- Business logic in HTTP handlers
- Direct database access from handlers
- Skipping authorization checks
- Editing generated code in gen/ directory
- Hardcoding configuration values
- Bypassing the OpenAPI-first workflow
- Creating new error types instead of using pkg/apperror

## 6. Positive Patterns to Encourage

### Go Best Practices

- Early returns to reduce nesting
- Table-driven tests for multiple scenarios
- Using context for cancellation and timeouts
- Dependency injection for testability
- Interfaces for abstraction at boundaries
- Proper use of pointer vs. value receivers

### Project Best Practices

- Following the domain module pattern consistently
- Comprehensive error handling with context
- Clear separation of concerns (handler → service → repository)
- Proper transaction handling for multi-step operations
- Using middleware for cross-cutting concerns
- Structured logging with request context
- Integration tests that verify the full request flow
- Pass go context through function calls

## 7. Review Feedback Guidelines

When providing feedback:

1. **Be Specific**: Point to exact files and line numbers
2. **Explain Why**: Don't just say "change this", explain the reasoning
3. **Provide Examples**: Show better alternatives with code snippets
4. **Prioritize**: Use severity labels (🔴 Critical, 🟠 High, 🟡 Medium, 🟢 Low)
5. **Be Constructive**: Frame feedback as improvement suggestions
6. **Acknowledge Good Work**: Call out well-implemented patterns

### Example Good Feedback

```markdown
🟠 High: Authorization Check Missing

File: pkg/game/service.go:45

The UpdateGame function doesn't verify that the user owns the game before updating it.

Suggested fix:

```go
func (s *Service) UpdateGame(ctx context.Context, userID string, gameID uuid.UUID, req UpdateGameRequest) error {
    // Verify ownership
    if err := s.repo.VerifyGameOwnership(ctx, gameID, userID); err != nil {
        return apperror.NewUnauthorizedError("user does not own this game")
    }
    // ... rest of update logic
}
```

This prevents users from modifying games they don't own.

```markdown
## 8. Quick Reference

### Before Submitting PR

```bash
make lint # Run all linters
make test # Run all tests
make build # Verify it builds
git status # Check no unintended files
```

### Self-Review Checklist

- [ ] Diff reviewed for unintended changes
- [ ] Tests added/updated
- [ ] Linter passes
- [ ] No debug code or print statements
- [ ] Error handling complete
- [ ] Documentation updated
- [ ] OpenAPI spec updated if API changed

### Reviewer Quick Checks

- [ ] PR description explains what and why
- [ ] CI/CD checks passing
- [ ] Code follows project patterns
- [ ] Authorization and validation present
- [ ] Tests cover new functionality
- [ ] No security vulnerabilities
- [ ] Performance considerations addressed

---

**Note**: This checklist is a guide, not a bureaucratic obstacle. Use judgment to apply these standards appropriately
based on the scope and impact of the changes.
