package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

type requestLoggingContextKey string

const requestKey requestLoggingContextKey = "requestLogging"

type Request struct {
	Method string
	Path   string
	ID     uuid.UUID
}

func RequestFromContext(ctx context.Context) (*Request, bool) {
	requestLogging, ok := ctx.Value(requestKey).(*Request)
	return requestLogging, ok
}

func RequestLogging(logLevel slog.Level) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requestLoggingContext := Request{
				Method: request.Method,
				Path:   request.RequestURI,
				ID:     uuid.New(),
			}

			ctx := context.WithValue(request.Context(), requestKey, requestLoggingContext)

			slog.Log(ctx, logLevel, "Incoming request", ":method", requestLoggingContext.Method, ":path", requestLoggingContext.Path, ":id", requestLoggingContext.ID)
			slog.Log(ctx, logLevel, "Request headers", ":headers", request.Header)

			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}
