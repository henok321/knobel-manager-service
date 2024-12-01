package middleware

import (
	"log/slog"
	"net/http"
	"regexp"
)

var ignorePattern = regexp.MustCompile("^/?(health|metrics)/?$")

func RequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		path := request.URL.Path

		if ignorePattern.Match([]byte(path)) {
			slog.Debug("Incomming request", "Method", request.Method, "Path", path)
		} else {
			slog.Info("Incomming request", "Method", request.Method, "Path", path)
		}
		next.ServeHTTP(writer, request)
	})
}
