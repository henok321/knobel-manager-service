package integrationtests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestGameSetup(t *testing.T) {
	tests := map[string]testCase{
		"Setup game tables": {
			method:             "POST",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusNoContent,
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
				_, err := db.ExecContext(t.Context(), "UPDATE games SET status = 'in_progress' WHERE id = 1")
				if err != nil {
					t.Fatalf("Failed to update game status: %v", err)
				}
			},
		},
		"Try to setup game with invalid number of teams": {
			method:             "POST",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusConflict,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Reset game setup": {
			method:             "DELETE",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusNoContent,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
			assertions: assertMatchmakingReset,
		},
		"Reset game setup without assigned tables": {
			method:             "DELETE",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusNoContent,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
			},
			assertions: assertMatchmakingReset,
		},
		"Reset game setup without permission": {
			method:             "DELETE",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusForbidden,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
			assertions: assertMatchmakingIntact,
		},
		"Reset game setup not in setup state": {
			method:             "DELETE",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"error":"Game is not in setup state"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned.sql")
			},
			assertions: assertMatchmakingIntact,
		},
	}

	dbConn, teardownDatabase := setupTestDatabase(t)
	defer teardownDatabase()

	db, err := sql.Open("pgx", dbConn)
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

	db, err := sql.Open("pgx", dbConn)
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

	// Setup is idempotent at the API level: it returns 204 each time
	// (it deletes and recreates rounds/tables on every call).
	tc := testCase{
		method:             "POST",
		endpoint:           "/games/1/setup",
		expectedStatusCode: http.StatusNoContent,
		requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
	}

	t.Run("First setup", func(t *testing.T) {
		newTestRequest(t, tc, server, db)
	})

	t.Run("Second setup", func(t *testing.T) {
		newTestRequest(t, tc, server, db)
	})

	t.Run("Third setup", func(t *testing.T) {
		newTestRequest(t, tc, server, db)
	})
}

func TestRosterChangeAfterSetup(t *testing.T) {
	dbConn, teardownDatabase := setupTestDatabase(t)
	defer teardownDatabase()

	db, err := sql.Open("pgx", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	runGooseUp(t, db)

	server, teardown := setupTestServer(t)
	defer teardown(server)

	executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
	advanceSequences(t, db)

	defer executeSQLFile(t, db, "./test_data/cleanup.sql")

	authorized := map[string]string{"Authorization": "Bearer sub-1"}

	steps := []struct {
		name string
		tc   testCase
	}{
		{"Assign tables", testCase{
			method:             "POST",
			endpoint:           "/games/1/setup",
			requestHeaders:     authorized,
			expectedStatusCode: http.StatusNoContent,
			assertions:         assertMatchmakingIntact,
		}},
		{"Adding a team is rejected while tables are assigned", testCase{
			method:             "POST",
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 9"}`,
			requestHeaders:     authorized,
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Game setup already assigned, reset the setup first"}`,
			assertions:         assertMatchmakingIntact,
		}},
		{"Reset the setup", testCase{
			method:             "DELETE",
			endpoint:           "/games/1/setup",
			requestHeaders:     authorized,
			expectedStatusCode: http.StatusNoContent,
			assertions:         assertMatchmakingReset,
		}},
		{"Add the team", testCase{
			method:             "POST",
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 9"}`,
			requestHeaders:     authorized,
			expectedStatusCode: http.StatusCreated,
			expectedBody:       `{"team": {"id":9,"name":"Team 9", "gameID":1}}`,
		}},
		{"Assign tables again", testCase{
			method:             "POST",
			endpoint:           "/games/1/setup",
			requestHeaders:     authorized,
			expectedStatusCode: http.StatusNoContent,
			assertions:         assertMatchmakingIntact,
		}},
	}

	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			newTestRequest(t, step.tc, server, db)
		})
	}
}
