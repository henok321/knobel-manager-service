package health

import "context"

type Checker interface {
	Name() string

	Check(ctx context.Context) error
}

type CheckResult struct {
	Name    string
	Status  string
	Message string
}

type CheckResults struct {
	Status string
	Checks map[string]CheckResult
}
