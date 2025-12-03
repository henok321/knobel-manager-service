package integrationtests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/lib/pq"
)

func TestTables(t *testing.T) {
	tests := map[string]testCase{
		"get tables for gameID 1 and round number 1": {
			method:             "GET",
			endpoint:           "/games/1/rounds/1/tables",
			expectedStatusCode: http.StatusOK,
			expectedBody:       readContentFromFile(t, "./test_data/round_1_tables.json"),

			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
		},
		"get tables for gameID 1 and round number 1 with scores": {
			method:             "GET",
			endpoint:           "/games/1/rounds/1/tables",
			expectedStatusCode: http.StatusOK,
			expectedBody:       readContentFromFile(t, "./test_data/round_1_tables_scores.json"),

			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned_with_scores.sql")
			},
		},
		"get tables for gameID 1 and round number 2": {
			method:             "GET",
			endpoint:           "/games/1/rounds/2/tables",
			expectedStatusCode: http.StatusOK,
			expectedBody:       readContentFromFile(t, "./test_data/round_2_tables.json"),
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
		},
		"get tables for game 1 and round number 1 forbidden": {
			method:             "GET",
			endpoint:           "/games/1/rounds/1/tables",
			expectedStatusCode: http.StatusForbidden,

			requestHeaders: map[string]string{"Authorization": "Bearer sub-2"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
		},
		"get tables game not found": {
			method:             "GET",
			endpoint:           "/games/2/rounds/1/tables",
			expectedStatusCode: http.StatusNotFound,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
		},
		"get tables round not found": {
			method:             "GET",
			endpoint:           "/games/1/rounds/3/tables",
			expectedStatusCode: http.StatusNotFound,

			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
		},
		"get tables invalid round number": {
			method:             "GET",
			endpoint:           "/games/1/rounds/invalid/tables",
			expectedStatusCode: http.StatusBadRequest,

			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
		},
		"get tables invalid game id": {
			method:             "GET",
			endpoint:           "/games/invalid/rounds/1/tables",
			expectedStatusCode: http.StatusBadRequest,

			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
		},
		"get table for gameID 1 and round number 2 by number 1": {
			method:             "GET",
			endpoint:           "/games/1/rounds/1/tables/1",
			expectedStatusCode: http.StatusOK,
			expectedBody:       readContentFromFile(t, "./test_data/round_1_tables_1.json"),
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
		},
		"get table by number game forbidden": {
			method:             "GET",
			endpoint:           "/games/1/rounds/1/tables/1",
			expectedStatusCode: http.StatusForbidden,

			requestHeaders: map[string]string{"Authorization": "Bearer sub-2"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
		},
		"get table by number game not found": {
			method:             "GET",
			endpoint:           "/games/2/rounds/1/tables/1",
			expectedStatusCode: http.StatusNotFound,

			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
		},
		"get table by id round not found": {
			method:             "GET",
			endpoint:           "/games/1/rounds/3/tables/1",
			expectedStatusCode: http.StatusNotFound,

			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
		},
		"get table by number table not found": {
			method:             "GET",
			endpoint:           "/games/1/rounds/1/tables/40",
			expectedStatusCode: http.StatusNotFound,

			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
		},
		"get table by number with invalid game id": {
			method:             "GET",
			endpoint:           "/games/invalid/rounds/1/tables/invalid",
			expectedStatusCode: http.StatusBadRequest,

			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
		},
		"get table by number with invalid round number": {
			method:             "GET",
			endpoint:           "/games/1/rounds/invalid/tables/1",
			expectedStatusCode: http.StatusBadRequest,

			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
		},
		"get table by number with invalid table number": {
			method:             "GET",
			endpoint:           "/games/1/rounds/1/tables/invalid",
			expectedStatusCode: http.StatusBadRequest,

			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
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
