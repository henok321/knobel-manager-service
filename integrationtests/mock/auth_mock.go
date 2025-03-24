package mock

import (
	"context"

	"firebase.google.com/go/v4/auth"
)

type FirebaseAuthMock struct {
}

func (m FirebaseAuthMock) VerifyIDToken(_ context.Context, idToken string) (*auth.Token, error) {

	return &auth.Token{
		UID:    idToken,
		Claims: map[string]any{"email": "test@example.org"},
	}, nil
}
