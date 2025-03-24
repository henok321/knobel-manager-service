package middleware

import (
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerName := strings.Trim(r.URL.Path, "/")
		if handlerName == "" {
			handlerName = "/"
		}

		duration := HTTPRequestDuration.MustCurryWith(prometheus.Labels{"handler": handlerName})
		counter := HTTPRequestsTotal.MustCurryWith(prometheus.Labels{"handler": handlerName})

		instrumentedHandler := promhttp.InstrumentHandlerDuration(duration, promhttp.InstrumentHandlerCounter(counter, next))

		instrumentedHandler.ServeHTTP(w, r)
	})
}
