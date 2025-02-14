package middleware

import (
	"context"
	"log/slog"
	"net/http"
)

func RequestLogging(logLevel slog.Level, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		slog.Log(context.Background(), logLevel, "Incoming request")
		next.ServeHTTP(writer, request)
	})
}
