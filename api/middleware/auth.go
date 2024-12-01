package middleware

import (
	"context"
	"net/http"
	"strings"

	firebase "firebase.google.com/go/v4"
)

type ContextKey string

const UserContextKey ContextKey = "user"

type User struct {
	Sub   string
	Email string
}

type AuthenticationMiddleware interface {
	Authentication(next http.Handler) http.Handler
}

type authenticationMiddleware struct {
	firebaseApp *firebase.App
}

func NewAuthenticationMiddleware(app *firebase.App) AuthenticationMiddleware {
	return authenticationMiddleware{
		firebaseApp: app,
	}
}

func (m authenticationMiddleware) Authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

		authorizationHeader := request.Header.Get("Authorization")

		if authorizationHeader == "" {
			http.Error(writer, `{"error": "forbidden"}`, http.StatusUnauthorized)
			return
		}

		tokenParts := strings.Split(authorizationHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			http.Error(writer, `{"error": "forbidden"}`, http.StatusUnauthorized)
			return
		}

		idToken := tokenParts[1]

		client, err := m.firebaseApp.Auth(request.Context())

		if err != nil {
			http.Error(writer, `{"error": "forbidden"}`, http.StatusInternalServerError)
			return
		}

		token, err := client.VerifyIDToken(request.Context(), idToken)

		if err != nil {
			http.Error(writer, `{"error": "forbidden"}`, http.StatusUnauthorized)
			return
		}

		userContext := &User{
			Sub:   token.UID,
			Email: token.Claims["email"].(string),
		}

		ctx := context.WithValue(request.Context(), UserContextKey, userContext)
		next.ServeHTTP(writer, request.WithContext(ctx))

	})
}
