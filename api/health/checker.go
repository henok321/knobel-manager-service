package health

import "context"

type Checker interface {
	Name() string

	Check(ctx context.Context) error
}

type CheckStatus string

const (
	CheckStatusPass CheckStatus = "pass"
	CheckStatusFail CheckStatus = "fail"
)

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDraining  Status = "draining"
)

type CheckResult struct {
	Name    string
	Status  CheckStatus
	Message string
}

type CheckResults struct {
	Status Status
	Checks map[string]CheckResult
}
