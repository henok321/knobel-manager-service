package health

import (
	"context"
	"sync/atomic"
)

type Service struct {
	checkers []Checker
	draining atomic.Bool
}

func NewService(checkers ...Checker) *Service {
	return &Service{
		checkers: checkers,
	}
}

func (s *Service) Readiness(ctx context.Context) CheckResults {
	if s.draining.Load() {
		return CheckResults{
			Status: StatusFail,
			Checks: make(map[string]CheckResult),
		}
	}

	results := CheckResults{
		Status: StatusPass,
		Checks: make(map[string]CheckResult),
	}

	for _, checker := range s.checkers {
		result := CheckResult{
			Name:   checker.Name(),
			Status: StatusPass,
		}

		if err := checker.Check(ctx); err != nil {
			result.Status = StatusFail
			result.Message = err.Error()
			results.Status = StatusFail
		}

		results.Checks[checker.Name()] = result
	}

	return results
}

func (s *Service) StartDraining() {
	s.draining.Store(true)
}
