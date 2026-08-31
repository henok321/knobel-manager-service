package mock

import (
	"context"
	"fmt"
	"strings"

	"firebase.google.com/go/v4/auth"
)

type FirebaseAuthMock struct{}

// VerifyIDToken treats the token string as the UID; the health-check probe token must fail.
func (m FirebaseAuthMock) VerifyIDToken(_ context.Context, idToken string) (*auth.Token, error) {
	if idToken == "health-check-invalid-token" {
		return nil, fmt.Errorf("invalid token")
	}

	return &auth.Token{
		UID:    idToken,
		Claims: map[string]any{"email": idToken + "@example.org"},
	}, nil
}

// GetUserByEmail fails for anything but "<uid>@example.org", as the real client does.
func (m FirebaseAuthMock) GetUserByEmail(_ context.Context, email string) (*auth.UserRecord, error) {
	uid, ok := strings.CutSuffix(email, "@example.org")
	if !ok || uid == "" || uid == "ghost" {
		return nil, fmt.Errorf("user not found: %s", email)
	}

	return userRecord(uid), nil
}

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
