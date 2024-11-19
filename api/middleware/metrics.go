package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	HttpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"handler", "method", "code"})

	HttpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"handler", "method", "code"})
)

func init() {
	prometheus.MustRegister(HttpRequestsTotal)
	prometheus.MustRegister(HttpRequestDuration)
}

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()

		handlerName := strings.Trim(c.FullPath(), "/")
		if handlerName == "" {
			handlerName = "/"
		}

		statusCode := c.Writer.Status()
		method := c.Request.Method

		labels := prometheus.Labels{
			"handler": handlerName,
			"method":  method,
			"code":    fmt.Sprintf("%d", statusCode),
		}

		HttpRequestsTotal.With(labels).Inc()
		HttpRequestDuration.With(labels).Observe(duration)
	}
}
