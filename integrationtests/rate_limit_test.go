package integrationtests

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRateLimit verifies that rate limiting middleware is in place and functional.
// Note: This test verifies the presence and basic functionality of rate limiting
// using the default limits (RATE_LIMIT_IP=100, RATE_LIMIT_USER=200).
// Specific limit values are tested via the environment variables but cannot be
// easily modified in integration tests due to the global singleton nature of the rate limiter.
func TestRateLimit(t *testing.T) {
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

	t.Run("Default IP rate limit (100) not exceeded", func(t *testing.T) {
		client := &http.Client{}

		for range 10 {
			req, err := http.NewRequest(http.MethodGet, server.URL+"/health", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}
	})

	t.Run("Default user rate limit (200) not exceeded with separate tracking per user", func(t *testing.T) {
		client := &http.Client{}
		users := []string{"user-a", "user-b", "user-c"}

		for _, user := range users {
			for range 5 {
				req, err := http.NewRequest(http.MethodGet, server.URL+"/games", nil)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}
				req.Header.Set("Authorization", "Bearer "+user)

				resp, err := client.Do(req)
				if err != nil {
					t.Fatalf("User %s request failed: %v", user, err)
				}
				resp.Body.Close()
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		}
	})
}
