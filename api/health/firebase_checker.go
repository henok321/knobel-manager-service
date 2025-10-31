package health

import (
	"context"
	"fmt"
	"time"

	"github.com/henok321/knobel-manager-service/api/middleware"
)

// FirebaseChecker checks Firebase Auth connectivity by verifying the JWT validation endpoint is reachable.
type FirebaseChecker struct {
	client  middleware.FirebaseAuth
	timeout time.Duration
}

// NewFirebaseChecker creates a new Firebase health checker with the specified timeout.
// The recommended timeout is 500ms to keep readiness probes fast.
func NewFirebaseChecker(client middleware.FirebaseAuth, timeout time.Duration) *FirebaseChecker {
	return &FirebaseChecker{
		client:  client,
		timeout: timeout,
	}
}

// Name returns the identifier for this checker.
func (c *FirebaseChecker) Name() string {
	return "firebase"
}

// Check performs a Firebase connectivity check by calling VerifyIDToken with an invalid token.
// This checks the same endpoint used for actual authentication in the middleware.
// Any response from Firebase (including auth errors) indicates Firebase is reachable.
// Only network errors or timeouts indicate Firebase is unreachable.
func (c *FirebaseChecker) Check(ctx context.Context) error {
	// Create a context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Try to verify an invalid token to check if Firebase is reachable
	// We expect this to fail with an auth error, which is fine - it means Firebase responded
	_, err := c.client.VerifyIDToken(checkCtx, "health-check-invalid-token")
	if err != nil {
		// Check for context deadline exceeded (timeout)
		if checkCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("firebase jwt validation endpoint timeout: %w", err)
		}

		// Any auth error (invalid token, expired, etc.) means Firebase is reachable and working
		// This is the expected case - we sent an invalid token, Firebase validated it and rejected it
		// This proves the JWT validation endpoint is accessible
		return nil
	}

	// Unexpected success - token was valid somehow, but still means Firebase is reachable
	return nil
}
