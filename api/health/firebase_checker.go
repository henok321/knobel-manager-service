package health

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/henok321/knobel-manager-service/api/middleware"
)

type FirebaseChecker struct {
	authClient middleware.FirebaseAuth
	timeout    time.Duration
}

func NewFirebaseChecker(authClient middleware.FirebaseAuth, timeout time.Duration) *FirebaseChecker {
	return &FirebaseChecker{
		authClient: authClient,
		timeout:    timeout,
	}
}

func (c *FirebaseChecker) Name() string {
	return "firebase"
}

func (c *FirebaseChecker) Check(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.authClient.VerifyIDToken(checkCtx, "health-check-invalid-token")
	if err != nil {
		if errors.Is(checkCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("firebase jwt validation endpoint timeout: %w", err)
		}

		return nil
	}

	return nil
}
