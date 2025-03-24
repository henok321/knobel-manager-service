package integrationtests

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	t.Run("health check", func(t *testing.T) {
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

		resp, err := http.Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to perform GET request: %v", err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")
	})
}
