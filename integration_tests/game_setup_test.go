package integration_tests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/lib/pq"
)

func TestGameSetup(t *testing.T) {
	tests := map[string]testCase{
		"Setup game tables": {
			method:             "POST",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusCreated,
			expectedHeaders:    map[string]string{"Location": "/games/1/tables"},
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
			},
		},
	}

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

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(db)
			}
			defer executeSQLFile(t, db, "./test_data/cleanup.sql")
			newTestRequest(t, tc, server)
		})
	}
}
