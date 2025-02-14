package middleware

import (
	"log/slog"
	"net/http"
	"regexp"
)

var ignorePattern = regexp.MustCompile("^/?(health|metrics)/?$")

func RequestLogging(logLevel slog.Level, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

		path := request.URL.Path

		requestGroup := slog.Group("request", "method", request.Method, "path", path)

		if ignorePattern.Match([]byte(path)) {
			slog.Debug("Handle request", requestGroup)
		} else {
			slog.Info("Handle request", requestGroup)
		}

		next.ServeHTTP(writer, request)
	})
}
