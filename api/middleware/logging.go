package middleware

import (
	"regexp"

	"github.com/gin-gonic/gin"

	log "github.com/sirupsen/logrus"
)

var ignorePattern = regexp.MustCompile("^/?(health|metrics)/?$")

func RequestLogging() gin.HandlerFunc {
	return func(c *gin.Context) {

		path := c.Request.URL.Path

		if ignorePattern.Match([]byte(path)) {
			log.Debugf("Request: %s %s", c.Request.Method, path)
		} else {
			log.Infof("Request: %s %s", c.Request.Method, path)
		}

		c.Next()
	}

}
