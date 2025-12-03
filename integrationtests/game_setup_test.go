package integrationtests

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
			expectedHeaders:    map[string]string{"Location": "/games/1"},
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
			},
		},
		"Try to setup game tables with out permissions": {
			method:             "POST",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusForbidden,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
			},
		},
		"Try to setup game not in setup state": {
			method:             "POST",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"error":"Game is not in setup state"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
				_, err := db.Exec("UPDATE games SET status = 'in_progress' WHERE id = 1")
				if err != nil {
					t.Fatalf("Failed to update game status: %v", err)
				}
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

	server, teardown := setupTestServer(t)
	defer teardown(server)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(db)
			}

			defer executeSQLFile(t, db, "./test_data/cleanup.sql")
			newTestRequest(t, tc, server, db)
		})
	}
}

func TestGameSetupMultipleTimes(t *testing.T) {
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

	// Setup test data
	executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
	defer executeSQLFile(t, db, "./test_data/cleanup.sql")

	tc := testCase{
		method:             "POST",
		endpoint:           "/games/1/setup",
		expectedStatusCode: http.StatusCreated,
		expectedHeaders:    map[string]string{"Location": "/games/1"},
		requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
	}

	// First setup - should succeed
	t.Run("First setup", func(t *testing.T) {
		newTestRequest(t, tc, server, db)
	})

	// Second setup - should also succeed (tests that reset works)
	t.Run("Second setup", func(t *testing.T) {
		newTestRequest(t, tc, server, db)
	})

	// Third setup - verify it can be run multiple times
	t.Run("Third setup", func(t *testing.T) {
		newTestRequest(t, tc, server, db)
	})
}
