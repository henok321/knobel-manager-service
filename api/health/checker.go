package health

import "context"

// Checker represents a health check that can be performed on a dependency.
// Implementations should be fast and time-bounded.
type Checker interface {
	// Name returns the unique identifier for this checker (e.g., "database", "redis", "s3")
	Name() string

	// Check performs the health check and returns an error if the dependency is unhealthy.
	// Implementations must respect the context and return quickly if the context is cancelled.
	Check(ctx context.Context) error
}

// CheckResult represents the result of a single health check.
type CheckResult struct {
	Name    string
	Status  string // "pass" or "fail"
	Message string // Optional error message if status is "fail"
}

// CheckResults represents the results of all health checks.
type CheckResults struct {
	Status string                 // "healthy", "unhealthy", or "draining"
	Checks map[string]CheckResult // Map of checker name to result
}
