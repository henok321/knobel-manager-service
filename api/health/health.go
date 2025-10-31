package health

import (
	"context"
	"sync/atomic"
)

// Service manages health checks for liveness and readiness probes.
type Service struct {
	checkers []Checker
	draining atomic.Bool
}

// NewService creates a new health service with the provided checkers.
func NewService(checkers ...Checker) *Service {
	return &Service{
		checkers: checkers,
	}
}

// Liveness performs a liveness check.
// This always returns healthy if the process is running.
// No dependencies are checked to keep this cheap and fast.
func (s *Service) Liveness() CheckResults {
	return CheckResults{
		Status: "healthy",
		Checks: make(map[string]CheckResult),
	}
}

// Readiness performs a readiness check by running all registered checkers.
// Returns unhealthy if any checker fails or if the service is draining.
func (s *Service) Readiness(ctx context.Context) CheckResults {
	// If draining, immediately return draining status
	if s.draining.Load() {
		return CheckResults{
			Status: "draining",
			Checks: make(map[string]CheckResult),
		}
	}

	results := CheckResults{
		Status: "healthy",
		Checks: make(map[string]CheckResult),
	}

	// Run all checkers
	for _, checker := range s.checkers {
		result := CheckResult{
			Name:   checker.Name(),
			Status: "pass",
		}

		if err := checker.Check(ctx); err != nil {
			result.Status = "fail"
			result.Message = err.Error()
			results.Status = "unhealthy"
		}

		results.Checks[checker.Name()] = result
	}

	return results
}

// StartDraining marks the service as draining.
// When draining, readiness checks will return unhealthy status.
// This allows load balancers and orchestrators to stop routing traffic
// to this instance before shutdown.
func (s *Service) StartDraining() {
	s.draining.Store(true)
}

// StopDraining marks the service as no longer draining.
// This can be used to re-enable readiness after a graceful shutdown was cancelled.
func (s *Service) StopDraining() {
	s.draining.Store(false)
}

// IsDraining returns true if the service is currently draining.
func (s *Service) IsDraining() bool {
	return s.draining.Load()
}
