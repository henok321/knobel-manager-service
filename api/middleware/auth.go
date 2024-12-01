package middleware

import (
	"context"
	"net/http"
	"strings"

	firebaseauth "github.com/henok321/knobel-manager-service/internal/auth"
)

type ContextKey string

const UserContextKey ContextKey = "user"

type User struct {
	Sub   string
	Email string
}

func Authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

		authorizationHeader := request.Header.Get("Authorization")

		if authorizationHeader == "" {
			http.Error(writer, `{"error": "forbidden"}`, http.StatusUnauthorized)
		}

		tokenParts := strings.Split(authorizationHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			http.Error(writer, `{"error": "forbidden"}`, http.StatusUnauthorized)
		}

		idToken := tokenParts[1]

		firebaseApp := firebaseauth.App

		client, err := firebaseApp.Auth(context.Background())

		if err != nil {
			http.Error(writer, `{"error": "forbidden"}`, http.StatusInternalServerError)
		}

		token, err := client.VerifyIDToken(context.Background(), idToken)

		if err != nil {
			http.Error(writer, `{"error": "forbidden"}`, http.StatusUnauthorized)
		}

		userContext := &User{
			Sub:   token.UID,
			Email: token.Claims["email"].(string),
		}

		ctx := context.WithValue(request.Context(), UserContextKey, userContext)
		next.ServeHTTP(writer, request.WithContext(ctx))

	})
}
