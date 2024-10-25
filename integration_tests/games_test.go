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
			name:               "GET games empty",
			method:             "GET",
			endpoint:           "/games",
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"games":[]}`,
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
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
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
		},
		{
			name:     "GET game by id ok",
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
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
		},
		{
			name:     "GET game by id not found",
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
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
		}, {
			name:     "GET game by id invalid id",
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
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
		}, {
			name:     "GET game by id not owner",
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
			requestHeaders:     map[string]string{"Authorization": "sub-2"},
		},
		{
			name:               "Create new game",
			method:             "POST",
			endpoint:           "/games",
			expectedStatusCode: http.StatusCreated,
			requestBody:        `{"name":"Game 1","numberOfRounds":2, "teamSize":4, "tableSize":4}`,
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
			expectedBody:       `{"game":{"id":1,"name":"Game 1","teamSize":4,"tableSize":4,"numberOfRounds":2,"status":"setup","owners":[{"gameID":1,"ownerSub":"sub-1"}]}}`,
			expectedHeaders:    map[string]string{"Location": "/games/1"},
		},
		{
			name:               "Create new game invalid request",
			method:             "POST",
			endpoint:           "/games",
			expectedStatusCode: http.StatusBadRequest,
			requestBody:        `{}`,
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
		},
		{
			name:               "Update an existing game",
			method:             "PUT",
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1 updated","numberOfRounds":3, "teamSize":4, "tableSize":4}`,
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"game":{"id":1,"name":"Game 1 updated","teamSize":4,"tableSize":4,"numberOfRounds":3,"status":"setup","owners":[{"gameID":1,"ownerSub":"sub-1"}]}}`,
			setup: func(db *sql.DB) {
				err := executeSQLFile(t, db, "./test_data/games_setup.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
		},
		{
			name:               "Update an existing game invalid request",
			method:             "PUT",
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1 updated","numberOfRounds":3}`,
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "Update an existing game not found",
			method:             "PUT",
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1 updated","numberOfRounds":3, "teamSize":4, "tableSize":4}`,
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "Update an existing game not owner",
			method:             "PUT",
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1 updated","numberOfRounds":3, "teamSize":4, "tableSize":4}`,
			requestHeaders:     map[string]string{"Authorization": "sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				err := executeSQLFile(t, db, "./test_data/games_setup.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
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

			if tc.setup != nil {
				tc.setup(db)
			}

			defer cleanupSetup(t, db, "./test_data/cleanup.sql")

			newTestRequest(t, tc, server)
		})
	}
}
