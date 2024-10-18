package integration_tests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/lib/pq"
)

func gamesTestCases(t *testing.T) []testCase {
	return []testCase{
		// GET /games when empty
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
		}, // GET /games with data
		{
			name:     "GET games by data",
			method:   "GET",
			endpoint: "/games",
			setup: func(db *sql.DB) {
				err := executeSQLFile(db, "../test_data/games/games_get_by_owner.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"games":[{"id":1,"name":"game 1","owners":[{"id":1,"sub":"sub-1"}]}]}`,
			headers:            map[string]string{"Authorization": "sub-1"},
		}, // POST /games to create a new game
		{
			name:     "POST create game",
			method:   "POST",
			endpoint: "/games",
			body:     `{"name":"new game"}`,
			setup: func(db *sql.DB) {
				// No initial setup needed
			},
			expectedStatusCode: http.StatusCreated,
			expectedBody:       `{"id":1,"name":"new game","owners":[{"id":1,"sub":"sub-1"}]}`,
			headers:            map[string]string{"Authorization": "sub-1", "Content-Type": "application/json"},
		}, // PUT /games/:id to update a game
		{
			name:     "PUT update game",
			method:   "PUT", // Assuming the game ID is 1
			endpoint: "/games/1",
			body:     `{"name":"updated game"}`,
			setup: func(db *sql.DB) {
				err := executeSQLFile(db, "../test_data/games/games_update.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"id":1,"name":"updated game","owners":[{"id":1,"sub":"sub-1"}]}`,
			headers:            map[string]string{"Authorization": "sub-1", "Content-Type": "application/json"},
		}, // DELETE /games/:id to delete a game
		{
			name:     "DELETE game",
			method:   "DELETE",
			endpoint: "/games/1",
			setup: func(db *sql.DB) {
				err := executeSQLFile(db, "../test_data/games/games_delete.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
			expectedStatusCode: http.StatusNoContent,
			expectedBody:       ``,
			headers:            map[string]string{"Authorization": "sub-1"},
		},
	}
}

func TestGames(t *testing.T) {

	tests := gamesTestCases(t)

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
