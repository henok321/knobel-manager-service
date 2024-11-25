package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	firebaseauth "github.com/henok321/knobel-manager-service/internal/auth"
)

type ContextKey string

const UserContextKey ContextKey = "user"

type User struct {
	Sub   string
	Email string
}

var ErrUserContextNotFound = errors.New("user context not found")

func GetUserFromCtx(ctx context.Context) (*User, error) {
	user, ok := ctx.Value(UserContextKey).(*User)
	if !ok {
		log.Warn("User context not found")
		return nil, ErrUserContextNotFound
	}
	return user, nil
}

func Authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

		authorizationHeader := request.Header.Get("Authorization")

		if authorizationHeader == "" {
			http.Error(writer, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		tokenParts := strings.Split(authorizationHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			http.Error(writer, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		idToken := tokenParts[1]

		firebaseApp := firebaseauth.App

		client, err := firebaseApp.Auth(context.Background())

		if err != nil {
			http.Error(writer, "Failed to initialize Firebase Auth client", http.StatusInternalServerError)
			return
		}

		token, err := client.VerifyIDToken(context.Background(), idToken)

		if err != nil {
			http.Error(writer, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		userContext := User{
			Sub:   token.UID,
			Email: token.Claims["email"].(string),
		}

		ctx := context.WithValue(request.Context(), UserContextKey, userContext)
		next.ServeHTTP(writer, request.WithContext(ctx))

	})
}
