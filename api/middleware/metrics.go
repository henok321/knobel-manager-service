package middleware

import (
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerName := strings.Trim(r.URL.Path, "/")
		if handlerName == "" {
			handlerName = "/"
		}

		duration := HttpRequestDuration.MustCurryWith(prometheus.Labels{"handler": handlerName})
		counter := HttpRequestsTotal.MustCurryWith(prometheus.Labels{"handler": handlerName})

		instrumentedHandler := promhttp.InstrumentHandlerDuration(duration, promhttp.InstrumentHandlerCounter(counter, next))

		instrumentedHandler.ServeHTTP(w, r)
	})
}
