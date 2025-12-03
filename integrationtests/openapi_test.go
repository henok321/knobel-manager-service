package integrationtests

import (
	"database/sql"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/assert/yaml"
	"github.com/stretchr/testify/require"
)

func TestOpenAPI(t *testing.T) {
	t.Run("openapi.yaml", func(t *testing.T) {
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

		resp, err := http.Get(server.URL + "/openapi.yaml")
		if err != nil {
			t.Fatalf("Failed to perform GET request: %v", err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

		openapiFile, err := os.ReadFile(filepath.Join("openapi", "openapi.yaml"))

		require.NoError(t, err)

		var openapiFileContent any
		var openapiResponseContent any

		err = yaml.Unmarshal(openapiFile, &openapiFileContent)
		require.NoError(t, err)

		responseData, err := io.ReadAll(resp.Body)

		require.NoError(t, err)

		err = yaml.Unmarshal(responseData, &openapiResponseContent)
		require.NoError(t, err)

		assert.Equal(t, openapiFileContent, openapiResponseContent)
	})

	t.Run("swagger docs", func(t *testing.T) {
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

		resp, err := http.Get(server.URL + "/docs")
		if err != nil {
			t.Fatalf("Failed to perform GET request: %v", err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
		require.NoError(t, err)
	})
}
