package integrationtests

import (
	"database/sql"
	"net/http"
	"os"
	"sync"
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
			req, err := http.NewRequest(http.MethodGet, server.URL+"/health/live", nil)
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
	err := os.Setenv("RATE_LIMIT_REQUESTS_PER_SECOND", "10")
	if err != nil {
		t.Fatalf("Failed to set RATE_LIMIT_REQUESTS_PER_SECOND: %v", err)
	}

	err = os.Setenv("RATE_LIMIT_BURST_SIZE", "10")
	if err != nil {
		t.Fatalf("Failed to set RATE_LIMIT_BURST_SIZE: %v", err)
	}

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

		wg := sync.WaitGroup{}

		for i := range 11 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, err := http.NewRequest(http.MethodGet, server.URL+"/health/live", nil)
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

				if i < 11 {
					assert.Equal(t, http.StatusOK, resp.StatusCode, "Request %d should succeed", i+1)
				} else {
					assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "Request %d should fail", i+1)
				}
			}()
		}
		wg.Wait()
	})
}
