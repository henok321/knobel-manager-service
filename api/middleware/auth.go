package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
)

type userContextKey string

const UserContextKey userContextKey = "user"

type FirebaseAuth interface {
	VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error)
}

type User struct {
	Sub   string
	Email string
}

func Authentication(authClient FirebaseAuth, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		authorizationHeader := request.Header.Get("Authorization")

		if authorizationHeader == "" {
			http.Error(writer, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}

		tokenParts := strings.Split(authorizationHeader, " ")

		requestContext := request.Context()

		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			slog.InfoContext(requestContext, "Malformed token")
			http.Error(writer, `{"error": "unauthorized"}`, http.StatusUnauthorized)

			return
		}

		idToken := tokenParts[1]

		token, err := authClient.VerifyIDToken(requestContext, idToken)
		if err != nil {
			slog.InfoContext(requestContext, "Invalid token", "error", err)
			http.Error(writer, `{"error": "unauthorized"}`, http.StatusUnauthorized)

			return
		}

		userContext := &User{
			Sub:   token.UID,
			Email: token.Claims["email"].(string),
		}

		ctx := context.WithValue(requestContext, UserContextKey, userContext)

		slog.InfoContext(ctx, "Request authenticated")

		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}
