package integration_tests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/lib/pq"
)

func playersTestCases(t *testing.T) []testCase {
	return []testCase{
		{
			name:     "GET players by gameID",
			method:   "GET",
			endpoint: "/games/1/players",
			setup: func(db *sql.DB) {
				err := executeSQLFile(db, "../test_data/players/players_get_by_game.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"players":[{"id":1,"name":"player 1","teamID":1},{"id":2,"name":"player 2","teamID":2}]}`,
			headers:            map[string]string{"Authorization": "sub-1", "Content-Type": "application/json"},
		},
	}
}

func TestPlayers(t *testing.T) {

	tests := playersTestCases(t)

	dbConn, teardownDatabase, _ := setupTestDatabase()
	defer teardownDatabase()

	db, _ := sql.Open("postgres", dbConn)
	defer db.Close()

	_ = runGooseUp(db)

	server, teardown := setupTestServer()
	defer teardown(server)

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {

			tc.setup(db)
			defer cleanupSetup(db, "../test_data/cleanup.sql")

			newTestRequest(t, tc, server)
		})
	}
}
