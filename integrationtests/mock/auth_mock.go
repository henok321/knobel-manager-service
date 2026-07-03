package mock

import (
	"context"
	"fmt"
	"strings"

	"firebase.google.com/go/v4/auth"
)

type FirebaseAuthMock struct{}

// VerifyIDToken mocks Firebase token verification.
// For health checks, it returns an error for invalid tokens (expected behavior).
// For test tokens, the token string is used as the UID and "<uid>@example.org" as email.
func (m FirebaseAuthMock) VerifyIDToken(_ context.Context, idToken string) (*auth.Token, error) {
	// If it's the health check token, return an error (Firebase would reject invalid tokens)
	if idToken == "health-check-invalid-token" {
		return nil, fmt.Errorf("invalid token")
	}

	return &auth.Token{
		UID:    idToken,
		Claims: map[string]any{"email": idToken + "@example.org"},
	}, nil
}

// GetUserByEmail resolves "<uid>@example.org" -> uid. Anything else is treated as not found,
// mirroring the error the real Firebase client returns for an unknown email.
func (m FirebaseAuthMock) GetUserByEmail(_ context.Context, email string) (*auth.UserRecord, error) {
	uid, ok := strings.CutSuffix(email, "@example.org")
	if !ok || uid == "" || uid == "ghost" {
		return nil, fmt.Errorf("user not found: %s", email)
	}

	return userRecord(uid), nil
}

// GetUsers echoes back a record per requested UID identifier.
func (m FirebaseAuthMock) GetUsers(_ context.Context, identifiers []auth.UserIdentifier) (*auth.GetUsersResult, error) {
	result := &auth.GetUsersResult{}

	for _, id := range identifiers {
		if uid, ok := id.(auth.UIDIdentifier); ok {
			result.Users = append(result.Users, userRecord(uid.UID))
		}
	}

	return result, nil
}

func userRecord(uid string) *auth.UserRecord {
	return &auth.UserRecord{UserInfo: &auth.UserInfo{UID: uid, Email: uid + "@example.org"}}
}
