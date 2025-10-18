package integrationtests

import (
	"database/sql"
	"net/http"
	"os"
	"testing"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit(t *testing.T) {
	// Reset rate limiter to ensure this test is isolated from others
	middleware.ResetRateLimitForTesting()

	os.Setenv("RATE_LIMIT_IP", "50")
	os.Setenv("RATE_LIMIT_USER", "10")

	defer func() {
		os.Unsetenv("RATE_LIMIT_IP")
		os.Unsetenv("RATE_LIMIT_USER")
		middleware.ResetRateLimitForTesting()
	}()

	dbConn, teardownDatabase := setupTestDatabase(t)
	defer teardownDatabase()

	db, err := sql.Open("postgres", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	runGooseUp(t, db)

	server, teardown := setupTestServer()
	defer teardown(server)

	t.Run("IP rate limit not exceeded with multiple requests", func(t *testing.T) {
		client := &http.Client{}

		for i := range 10 {
			req, err := http.NewRequest(http.MethodGet, server.URL+"/health", nil)
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

	t.Run("User rate limit not exceeded with separate tracking per user", func(t *testing.T) {
		client := &http.Client{}
		users := []string{"user-a", "user-b", "user-c"}

		for _, user := range users {
			for i := range 5 {
				req, err := http.NewRequest(http.MethodGet, server.URL+"/games", nil)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}
				req.Header.Set("Authorization", "Bearer "+user)

				resp, err := client.Do(req)
				if err != nil {
					t.Fatalf("User %s request %d failed: %v", user, i+1, err)
				}
				resp.Body.Close()
				assert.Equal(t, http.StatusOK, resp.StatusCode, "User %s request %d should succeed", user, i+1)
			}
		}
	})

	t.Run("User rate limit exceeded when limit is 10", func(t *testing.T) {
		client := &http.Client{}
		user := "test-user"

		for i := range 11 {
			req, err := http.NewRequest(http.MethodGet, server.URL+"/games", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Authorization", "Bearer "+user)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request %d failed: %v", i+1, err)
			}
			resp.Body.Close()

			if i < 10 {
				assert.Equal(t, http.StatusOK, resp.StatusCode, "Request %d should succeed", i+1)
			} else {
				assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "Request %d should be rate limited", i+1)
			}
		}
	})
}
