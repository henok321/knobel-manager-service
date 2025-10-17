package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
)

type userContextKey string

const userKey userContextKey = "user"

type FirebaseAuth interface {
	VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error)
}

type User struct {
	Sub   string
	Email string
}

func UserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userKey).(*User)
	return user, ok
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

		// Safely extract email claim with type assertion to prevent panic
		email, _ := token.Claims["email"].(string)

		userContext := &User{
			Sub:   token.UID,
			Email: email,
		}

		ctx := context.WithValue(requestContext, userKey, userContext)

		slog.InfoContext(ctx, "Request authenticated")

		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}
