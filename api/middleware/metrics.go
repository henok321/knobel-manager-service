package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	HTTPRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"handler", "method", "code"})

	HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"handler", "method", "code"})
)

func init() {
	prometheus.MustRegister(HTTPRequestsTotal)
	prometheus.MustRegister(HTTPRequestDuration)
}

type recordingWriter struct {
	http.ResponseWriter
	code int
}

func (w *recordingWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func Metrics() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Route template, not r.URL.Path: raw paths carry unbounded IDs and let anyone OOM us via metric cardinality.
			handlerName := r.Pattern
			if handlerName == "" {
				handlerName = "unmatched"
			}

			recorder := &recordingWriter{ResponseWriter: w, code: http.StatusOK}
			start := time.Now()

			defer func() {
				method, code := strings.ToLower(r.Method), strconv.Itoa(recorder.code)
				HTTPRequestsTotal.WithLabelValues(handlerName, method, code).Inc()
				HTTPRequestDuration.WithLabelValues(handlerName, method, code).Observe(time.Since(start).Seconds())
			}()

			next.ServeHTTP(recorder, r)
		})
	}
}
