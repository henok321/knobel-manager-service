package middleware

import (
	"log/slog"
	"net/http"
)

type RequestLoggingMiddleware interface {
	RequestLogging(next http.Handler) http.Handler
}

type requestLoggingMiddleware struct {
	logLevel slog.Level
}

func NewRequestLoggingMiddleware(level slog.Level) RequestLoggingMiddleware {
	return requestLoggingMiddleware{logLevel: level}
}

func (m requestLoggingMiddleware) RequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch m.logLevel {
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
