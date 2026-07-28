package health

import (
	"context"
	"sync/atomic"

	"github.com/henok321/knobel-manager-service/gen/health"
)

type check = struct {
	Message *string                                        `json:"message,omitempty"`
	Status  health.HealthCheckDetailedResponseChecksStatus `json:"status"`
}

type Service struct {
	checkers []Checker
	draining atomic.Bool
}

func NewService(checkers ...Checker) *Service {
	return &Service{
		checkers: checkers,
	}
}

func (s *Service) Readiness(ctx context.Context) health.HealthCheckDetailedResponse {
	if s.draining.Load() {
		return health.HealthCheckDetailedResponse{Status: health.HealthCheckDetailedResponseStatusFail}
	}

	checks := map[string]check{}
	status := health.HealthCheckDetailedResponseStatusPass

	for _, checker := range s.checkers {
		result := check{Status: health.HealthCheckDetailedResponseChecksStatusPass}

		if err := checker.Check(ctx); err != nil {
			message := err.Error()
			result.Status = health.HealthCheckDetailedResponseChecksStatusFail
			result.Message = &message
			status = health.HealthCheckDetailedResponseStatusFail
		}

		checks[checker.Name()] = result
	}

	response := health.HealthCheckDetailedResponse{Status: status}
	if len(checks) > 0 {
		response.Checks = &checks
	}

	return response
}

func (s *Service) StartDraining() {
	s.draining.Store(true)
}
