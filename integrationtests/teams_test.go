package integrationtests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
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
		"Create team while the tables are assigned": {
			method:             "POST",
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 9"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Game setup already assigned, reset the setup first"}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
				advanceSequences(t, db)
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()
				assert.Equal(t, 8, countRows(t, db, "SELECT COUNT(*) FROM teams WHERE game_id = 1"), "team must not be created")
				assertTablesAssigned(t, db)
			},
		},
		"Create team in a running game": {
			method:             "POST",
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 9"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Game is not editable"}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned.sql")
				advanceSequences(t, db)
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()
				assert.Equal(t, 8, countRows(t, db, "SELECT COUNT(*) FROM teams WHERE game_id = 1"), "team must not be created")
				assertTablesAssigned(t, db)
			},
		},
		"Delete team while the tables are assigned": {
			method:             "DELETE",
			endpoint:           "/games/1/teams/8",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Game setup already assigned, reset the setup first"}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()
				assert.Equal(t, 8, countRows(t, db, "SELECT COUNT(*) FROM teams WHERE game_id = 1"), "team must not be deleted")
				assertTablesAssigned(t, db)
			},
		},
		"Delete team in a running game": {
			method:             "DELETE",
			endpoint:           "/games/1/teams/8",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Game is not editable"}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()
				assert.Equal(t, 8, countRows(t, db, "SELECT COUNT(*) FROM teams WHERE game_id = 1"), "team must not be deleted")
				assertTablesAssigned(t, db)
			},
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
