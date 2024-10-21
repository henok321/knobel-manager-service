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
	}
}

func TestGames(t *testing.T) {

	tests := gamesTestCases(t)

	dbConn, teardownDatabase, _ := setupTestDatabase(t)
	defer teardownDatabase()

	db, _ := sql.Open("postgres", dbConn)
	defer db.Close()

	_ = runGooseUp(t, db)

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
