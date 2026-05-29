package integrationtests

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRateLimitNotExceededWithDefaults(t *testing.T) {
	dbConn, teardownDatabase := setupTestDatabase(t)
	defer teardownDatabase()

	db, err := sql.Open("postgres", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	runGooseUp(t, db)

	server, teardown := setupTestServer(t)
	defer teardown(server)

	t.Run("IP rate limit not exceeded with multiple requests", func(t *testing.T) {
		client := &http.Client{}

		for i := range 10 {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/health/live", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request %d failed: %v", i+1, err)
			}
			resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode, "Request %d should succeed", i+1)
		}
	})
}

func TestRateLimitExceededWithEnv(t *testing.T) {
	t.Setenv("RATE_LIMIT_REQUESTS_PER_SECOND", "10")

	t.Setenv("RATE_LIMIT_BURST_SIZE", "10")

	defer func() {
		err := os.Unsetenv("RATE_LIMIT")
		if err != nil {
			t.Fatalf("Failed to unset RATE_LIMIT: %v", err)
		}
	}()

	dbConn, teardownDatabase := setupTestDatabase(t)
	defer teardownDatabase()

	db, err := sql.Open("postgres", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	runGooseUp(t, db)

	server, teardown := setupTestServer(t)
	defer teardown(server)

	t.Run("IP rate limit exceeded with multiple requests", func(t *testing.T) {
		client := &http.Client{}

		const total = 15
		var ok, tooMany atomic.Int32
		wg := sync.WaitGroup{}

		for i := range total {
			wg.Go(func() {
				req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/health/live", nil)
				if err != nil {
					t.Error("Failed to create request", err)
					return
				}

				resp, err := client.Do(req)
				if err != nil {
					t.Errorf("Request %d failed: %v", i+1, err)
					return
				}
				defer resp.Body.Close()

				switch resp.StatusCode {
				case http.StatusOK:
					ok.Add(1)
				case http.StatusTooManyRequests:
					tooMany.Add(1)
				default:
					t.Errorf("Request %d returned unexpected status %d", i+1, resp.StatusCode)
				}
			})
		}
		wg.Wait()

		// Burst is 10, so up to 10 should succeed and the rest should be rejected.
		// Concurrent timing means exact split varies, but we expect at least one
		// 429 and the success count to not exceed the burst.
		assert.Positive(t, int(tooMany.Load()), "at least one request should exceed the limit")
		assert.LessOrEqual(t, int(ok.Load()), 10, "successful requests must not exceed burst size")
	})
}

func TestRateLimitXFFRotationDoesNotBypassWhenUntrusted(t *testing.T) {
	t.Setenv("RATE_LIMIT_REQUESTS_PER_SECOND", "5")
	t.Setenv("RATE_LIMIT_BURST_SIZE", "5")

	dbConn, teardownDatabase := setupTestDatabase(t)
	defer teardownDatabase()

	db, err := sql.Open("postgres", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	runGooseUp(t, db)

	server, teardown := setupTestServer(t)
	defer teardown(server)

	client := &http.Client{}

	const total = 20
	var tooMany atomic.Int32
	wg := sync.WaitGroup{}

	for i := range total {
		wg.Go(func() {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/health/live", nil)
			if err != nil {
				t.Errorf("Failed to create request: %v", err)
				return
			}
			req.Header.Set("X-Forwarded-For", fmt.Sprintf("203.0.113.%d", i+1))

			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("Request %d failed: %v", i+1, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusTooManyRequests {
				tooMany.Add(1)
			}
		})
	}
	wg.Wait()

	// All 20 requests share RemoteAddr (127.0.0.1) because XFF is not trusted.
	// With burst 5, at least 10 must be rejected. If XFF were trusted, each unique
	// IP would get its own bucket and zero requests would be rejected.
	assert.GreaterOrEqual(t, int(tooMany.Load()), 10, "XFF rotation must not bypass per-IP limit when TRUST_FORWARDED_FOR is unset")
}

func TestRateLimitTrustsRightmostXFFWhenTrusted(t *testing.T) {
	t.Setenv("RATE_LIMIT_REQUESTS_PER_SECOND", "5")
	t.Setenv("RATE_LIMIT_BURST_SIZE", "5")
	t.Setenv("TRUST_FORWARDED_FOR", "true")

	dbConn, teardownDatabase := setupTestDatabase(t)
	defer teardownDatabase()

	db, err := sql.Open("postgres", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	runGooseUp(t, db)

	server, teardown := setupTestServer(t)
	defer teardown(server)

	client := &http.Client{}

	const total = 20
	var tooMany atomic.Int32
	wg := sync.WaitGroup{}

	for i := range total {
		wg.Go(func() {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/health/live", nil)
			if err != nil {
				t.Errorf("Failed to create request: %v", err)
				return
			}
			// Vary the leftmost (client-controlled) entry; pin the rightmost
			// (proxy-observed) entry — this is the rate-limit key.
			req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d, 203.0.113.7", i+1))

			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("Request %d failed: %v", i+1, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusTooManyRequests {
				tooMany.Add(1)
			}
		})
	}
	wg.Wait()

	// All 20 requests collapse onto the rightmost XFF entry (203.0.113.7).
	// With burst 5, at least 10 must be rejected. Spoofing the leftmost entry
	// must not yield a fresh bucket.
	assert.GreaterOrEqual(t, int(tooMany.Load()), 10, "client-controlled (leftmost) XFF must not influence the rate-limit key when proxy is trusted")
}
