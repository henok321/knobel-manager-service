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

func (s *Service) Liveness() CheckResults {
	return CheckResults{
		Status: StatusHealthy,
		Checks: make(map[string]CheckResult),
	}
}

func (s *Service) Readiness(ctx context.Context) CheckResults {
	if s.draining.Load() {
		return CheckResults{
			Status: StatusDraining,
			Checks: make(map[string]CheckResult),
		}
	}

	results := CheckResults{
		Status: StatusHealthy,
		Checks: make(map[string]CheckResult),
	}

	for _, checker := range s.checkers {
		result := CheckResult{
			Name:   checker.Name(),
			Status: CheckStatusPass,
		}

		if err := checker.Check(ctx); err != nil {
			result.Status = CheckStatusFail
			result.Message = err.Error()
			results.Status = StatusUnhealthy
		}

		results.Checks[checker.Name()] = result
	}

	return results
}

func (s *Service) StartDraining() {
	s.draining.Store(true)
}
