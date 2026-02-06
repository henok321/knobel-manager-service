package health

import "context"

type Checker interface {
	Name() string

	Check(ctx context.Context) error
}

type ComponentStatus string

const (
	ComponentStatusPass ComponentStatus = "pass"
	ComponentStatusFail ComponentStatus = "fail"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusFail Status = "fail"
)

type CheckResult struct {
	Name    string
	Status  ComponentStatus
	Message string
}

type CheckResults struct {
	Status Status
	Checks map[string]CheckResult
}
