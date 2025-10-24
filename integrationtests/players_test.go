package integrationtests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/lib/pq"
)

func TestPlayers(t *testing.T) {
	tests := map[string]testCase{
		"Create player": {
			method:             "POST",
			endpoint:           "/games/1/teams/1/players",
			requestBody:        `{"name":"Player 1"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusCreated,
			expectedBody:       `{"player": {"id":1,"name":"Player 1", "teamID":1}}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
			},
		},
		"Create player invalid body": {
			method:             "POST",
			endpoint:           "/games/1/teams/1/players",
			requestBody:        `{"invalid":"Player 1"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusBadRequest,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
			},
		},
		"Create player not found": {
			method:             "POST",
			endpoint:           "/games/1/teams/1/players",
			requestBody:        `{"name":"Player 1"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Create player not the owner": {
			method:             "POST",
			endpoint:           "/games/1/teams/1/players",
			requestBody:        `{"name":"Player 1"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
			},
		},
		"Update player": {
			method:             "PUT",
			endpoint:           "/games/1/teams/1/players/1",
			requestBody:        `{"name":"Player 1 Updated"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"player": {"id":1,"name":"Player 1 Updated","teamID": 1}}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team_player.sql")
			},
		},
		"Update player not found": {
			method:             "PUT",
			endpoint:           "/games/1/teams/1/players/1",
			requestBody:        `{"name":"Player 1 Updated"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
			},
		},
		"Update player forbidden": {
			method:             "PUT",
			endpoint:           "/games/1/teams/1/players/1",
			requestBody:        `{"name":"Player 1 Updated"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusNotFound,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
			},
		},
		"Delete player": {
			method:             "DELETE",
			endpoint:           "/games/1/teams/1/players/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team_player.sql")
			},
		},
		"Delete player not found": {
			method:             "DELETE",
			endpoint:           "/games/1/teams/1/players/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
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
			newTestRequest(t, tc, server, db)
		})
	}
}
