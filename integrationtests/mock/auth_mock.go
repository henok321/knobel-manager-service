package mock

import (
	"context"
	"fmt"

	"firebase.google.com/go/v4/auth"
)

type FirebaseAuthMock struct{}

// VerifyIDToken mocks Firebase token verification.
// For health checks, it returns an error for invalid tokens (expected behavior).
// For test tokens, it returns a valid token.
func (m FirebaseAuthMock) VerifyIDToken(_ context.Context, idToken string) (*auth.Token, error) {
	// If it's the health check token, return an error (Firebase would reject invalid tokens)
	if idToken == "health-check-invalid-token" {
		return nil, fmt.Errorf("invalid token")
	}

	// For actual test tokens, return a valid token
	return &auth.Token{
		UID:    idToken,
		Claims: map[string]any{"email": "test@example.org"},
	}, nil
}
