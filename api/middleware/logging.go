package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
)

type requestLoggingContextKey string

const requestKey requestLoggingContextKey = "requestLogging"

type Request struct {
	Method string
	Path   string
	ID     string
}

func RequestFromContext(ctx context.Context) (*Request, bool) {
	requestLogging, ok := ctx.Value(requestKey).(*Request)
	return requestLogging, ok
}

func RequestLogging(logLevel slog.Level) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			b := make([]byte, 16)
			_, _ = rand.Read(b)
			requestLoggingContext := Request{
				Method: request.Method,
				Path:   request.RequestURI,
				ID:     hex.EncodeToString(b),
			}

			ctx := context.WithValue(request.Context(), requestKey, requestLoggingContext)

			slog.Log(ctx, logLevel, "Incoming request")

			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}
