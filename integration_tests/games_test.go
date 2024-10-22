package integration_tests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/lib/pq"
)

func gamesTestCases(t *testing.T) []testCase {
	return []testCase{
		{
			name:     "GET games empty",
			method:   "GET",
			endpoint: "/games",
			setup: func(db *sql.DB) {
				// No setup needed; database is empty
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"games":[]}`,
			headers:            map[string]string{"Authorization": "sub-1"},
		},
		{
			name:     "GET games",
			method:   "GET",
			endpoint: "/games",
			setup: func(db *sql.DB) {
				err := executeSQLFile(t, db, "./test_data/games_setup.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       readContentFromFile(t, "./test_data/games_setup.json"),
			headers:            map[string]string{"Authorization": "sub-1"},
		},
		{
			name:     "GET game by id - ok",
			method:   "GET",
			endpoint: "/games/1",
			setup: func(db *sql.DB) {
				err := executeSQLFile(t, db, "./test_data/games_setup.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       readContentFromFile(t, "./test_data/games_setup_by_id.json"),
			headers:            map[string]string{"Authorization": "sub-1"},
		},
		{
			name:     "GET game by id - not found",
			method:   "GET",
			endpoint: "/games/2",
			setup: func(db *sql.DB) {
				err := executeSQLFile(t, db, "./test_data/games_setup.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
			expectedStatusCode: http.StatusNotFound,
			expectedBody:       `{"error":"game not found"}`,
			headers:            map[string]string{"Authorization": "sub-1"},
		}, {
			name:     "GET game by id - invalid id",
			method:   "GET",
			endpoint: "/games/invalid",
			setup: func(db *sql.DB) {
				err := executeSQLFile(t, db, "./test_data/games_setup.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"error":"invalid id"}`,
			headers:            map[string]string{"Authorization": "sub-1"},
		}, {
			name:     "GET game by id  - not owner",
			method:   "GET",
			endpoint: "/games/1",
			setup: func(db *sql.DB) {
				err := executeSQLFile(t, db, "./test_data/games_setup.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
			expectedStatusCode: http.StatusForbidden,
			expectedBody:       `{"error":"user is not the owner of the game"}`,
			headers:            map[string]string{"Authorization": "sub-2"},
		},
	}
}

func TestGames(t *testing.T) {

	tests := gamesTestCases(t)

	dbConn, teardownDatabase, _ := setupTestDatabase(t)
	defer teardownDatabase()

	db, err := sql.Open("postgres", dbConn)

	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}

	defer db.Close()

	err = runGooseUp(t, db)

	if err != nil {
		t.Fatalf("Failed to run goose up: %v", err)
	}

	server, teardown := setupTestServer()
	defer teardown(server)

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {

			tc.setup(db)
			defer cleanupSetup(t, db, "./test_data/cleanup.sql")

			newTestRequest(t, tc, server)
		})
	}
}
