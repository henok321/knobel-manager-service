package health

import "context"

type Checker interface {
	Name() string

	Check(ctx context.Context) error
}

type CheckResult struct {
	Name    string
	Status  string // "pass" or "fail"
	Message string // Optional error message if status is "fail"
}

type CheckResults struct {
	Status string                 // "healthy", "unhealthy", or "draining"
	Checks map[string]CheckResult // Map of checker name to result
}
