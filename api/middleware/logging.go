package middleware

import (
	"net/http"
	"regexp"

	log "github.com/sirupsen/logrus"
)

var ignorePattern = regexp.MustCompile("^/?(health|metrics)/?$")

func RequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		path := request.URL.Path

		if ignorePattern.Match([]byte(path)) {
			log.Debugf("Request: %s %s", request.Method, path)
		} else {
			log.Infof("Request: %s %s", request.Method, path)
		}
		next.ServeHTTP(writer, request)
	})
}
