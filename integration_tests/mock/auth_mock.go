package mock

import (
	"context"

	"firebase.google.com/go/v4/auth"
)

type MockFirebaseAuth struct {
}

func (m MockFirebaseAuth) VerifyIDToken(_ context.Context, idToken string) (*auth.Token, error) {

	return &auth.Token{
		UID:    idToken,
		Claims: map[string]interface{}{"email": "test@example.org"},
	}, nil
}
