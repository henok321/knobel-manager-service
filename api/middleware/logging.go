package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

type requestLoggingContextKey string

const RequestLoggingContext requestLoggingContextKey = "requestLogging"

type Request struct {
	Method string
	Path   string
	ID     uuid.UUID
}

func RequestLogging(logLevel slog.Level, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestLoggingContext := Request{
			Method: request.Method,
			Path:   request.RequestURI,
			ID:     uuid.New(),
		}

		ctx := context.WithValue(request.Context(), RequestLoggingContext, requestLoggingContext)

		slog.Log(ctx, logLevel, "Incoming request")

		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}
