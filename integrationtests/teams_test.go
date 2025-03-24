package integrationtests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/lib/pq"
)

func TestTeams(t *testing.T) {
	tests := map[string]testCase{
		"Create team": {
			method:             "POST",
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 1"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusCreated,
			expectedBody:       `{"team": {"id":1,"name":"Team 1", "gameID":1}}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Create team with players": {
			method:             "POST",
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 1","players": [{"name":"Player 1"},{"name":"Player 2"},{"name":"Player 3"},{"name":"Player 4"}]}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusCreated,
			expectedBody:       `{"team":{"id":1,"name":"Team 1","gameID":1,"players":[{"id":1,"name":"Player 1","teamID":1},{"id":2,"name":"Player 2","teamID":1},{"id":3,"name":"Player 3","teamID":1},{"id":4,"name":"Player 4","teamID":1}]}}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Create team with players invalid player count": {
			method:             "POST",
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 1","players": [{"name":"Player 1"},{"name":"Player 2"},{"name":"Player 3"},{"name":"Player 4"},{"name":"Player 5"}]}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusBadRequest,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Create team not owner": {
			method:             "POST",
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 1"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Update team": {
			method:             "PUT",
			endpoint:           "/games/1/teams/1",
			requestBody:        `{"name":"Team 1 updated"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"team": {"id":1,"name":"Team 1 updated", "gameID":1}}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
			},
		},
		"Update team invalid teamID": {
			method:             "PUT",
			endpoint:           "/games/1/teams/invalid",
			requestBody:        `{"name":"Team 1 updated"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusBadRequest,
		},
		"Update team invalid gameID": {
			method:             "PUT",
			endpoint:           "/games/invalid/teams/1",
			requestBody:        `{"name":"Team 1 updated"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusBadRequest,
		},
		"Update team not owner": {
			method:             "PUT",
			endpoint:           "/games/1/teams/1",
			requestBody:        `{"name":"Team 1 updated"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
			},
		},
		"Update team game not found": {
			method:             "PUT",
			endpoint:           "/games/2/teams/1",
			requestBody:        `{"name":"Team 1 updated"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
			},
		},
		"Update team not found": {
			method:             "PUT",
			endpoint:           "/games/1/teams/2",
			requestBody:        `{"name":"Team 2 updated"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
			},
		},
		"Delete team": {
			method:             "DELETE",
			endpoint:           "/games/1/teams/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
			},
		},
		"Delete team game not found": {
			method:             "DELETE",
			endpoint:           "/games/2/teams/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Delete team not owner": {
			method:             "DELETE",
			endpoint:           "/games/1/teams/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Delete team invalid gameID": {
			method:             "DELETE",
			endpoint:           "/games/invalid/teams/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusBadRequest,
		},
		"Delete team invalid teamID": {
			method:             "DELETE",
			endpoint:           "/games/1/teams/invalid",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusBadRequest,
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
