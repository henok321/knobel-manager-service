package middleware

import (
	"log/slog"
	"net/http"
)

func RequestLogging(logLevel slog.Level, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch logLevel {
		case slog.LevelDebug:
			slog.Debug("Incoming request", "method", request.Method, "path", request.URL.Path)
		case slog.LevelInfo:
			slog.Info("Incoming request", "method", request.Method, "path", request.URL.Path)
		case slog.LevelWarn:
			slog.Warn("Incoming request", "method", request.Method, "path", request.URL.Path)
		case slog.LevelError:
			slog.Error("Incmming request", "method", request.Method, "path", request.URL.Path)
		default:
			slog.Info("Incoming request", "method", request.Method, "path", request.URL.Path)
		}

		next.ServeHTTP(writer, request)
	})
}
