package integrationtests

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/henok321/knobel-manager-service/gen/health"
)

func TestHealthCheck(t *testing.T) {
	t.Run("liveness check pass", func(t *testing.T) {
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

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/health/live", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to perform GET request: %v", err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

		var response health.HealthCheckResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "pass", response.Status)
	})

	t.Run("readiness check pass", func(t *testing.T) {
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

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/health/ready", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to perform GET request: %v", err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

		var response health.HealthCheckDetailedResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, health.HealthCheckDetailedResponseStatus("pass"), response.Status)

		require.NotNil(t, response.Checks)
		checks := *response.Checks

		dbCheck, exists := checks["database"]
		require.True(t, exists, "Database check should be present")
		assert.Equal(t, health.HealthCheckDetailedResponseChecksStatus("pass"), dbCheck.Status)

		firebaseCheck, exists := checks["firebase"]
		require.True(t, exists, "Firebase check should be present")
		assert.Equal(t, health.HealthCheckDetailedResponseChecksStatus("pass"), firebaseCheck.Status)
	})

	t.Run("readiness check database not available fail", func(t *testing.T) {
		dbConn, teardownDatabase := setupTestDatabase(t)

		db, err := sql.Open("postgres", dbConn)
		if err != nil {
			t.Fatalf("Failed to open database connection: %v", err)
		}

		defer db.Close()

		runGooseUp(t, db)

		server, teardown := setupTestServer(t)
		defer teardown(server)

		// Stop database to simulate database not available
		teardownDatabase()

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/health/ready", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to perform GET request: %v", err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "Expected status code 503")

		var response health.HealthCheckDetailedResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, health.HealthCheckDetailedResponseStatus("fail"), response.Status)

		require.NotNil(t, response.Checks)
		checks := *response.Checks

		dbCheck, exists := checks["database"]
		require.True(t, exists, "Database check should be present")
		assert.Equal(t, health.HealthCheckDetailedResponseChecksStatus("fail"), dbCheck.Status)

		firebaseCheck, exists := checks["firebase"]
		require.True(t, exists, "Firebase check should be present")
		assert.Equal(t, health.HealthCheckDetailedResponseChecksStatus("pass"), firebaseCheck.Status)
	})
}
