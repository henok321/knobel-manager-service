package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMetricsLabelsUseRoutePatternNotRawPath(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /games/{gameID}", Metrics()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	for _, id := range []string{"1", "2", "3"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/games/"+id, nil)
		mux.ServeHTTP(httptest.NewRecorder(), req)
	}

	// promhttp lowercases the method label.
	patternSeries := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET /games/{gameID}", "get", "200"))
	assert.Equal(t, 3, int(patternSeries), "distinct IDs must collapse into one series keyed by the route template")

	rawPathSeries := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("games/1", "get", "200"))
	assert.Equal(t, 0, int(rawPathSeries), "raw request paths must never become metric label values")
}
