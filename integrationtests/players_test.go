package integrationtests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
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
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team_player.sql")
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
		"Delete player forbidden": {
			method:             "DELETE",
			endpoint:           "/games/1/teams/1/players/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team_player.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Equal(t, 1, countRows(t, db, "SELECT COUNT(*) FROM players WHERE id = 1"),
					"player must not be deleted by a non-owner")
			},
		},
		"Create player after matchmaking is rejected": {
			method:             "POST",
			endpoint:           "/games/1/teams/1/players",
			requestBody:        `{"name":"Player 33"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Game setup already assigned, reset the setup first"}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
				advanceSequences(t, db)
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()
				assert.Equal(t, 32, countRows(t, db, "SELECT COUNT(*) FROM players"), "player must not be created")
				assertMatchmakingIntact(t, db)
			},
		},
		"Create player in a running game": {
			method:             "POST",
			endpoint:           "/games/1/teams/1/players",
			requestBody:        `{"name":"Player 33"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Game is not editable"}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned.sql")
				advanceSequences(t, db)
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()
				assert.Equal(t, 32, countRows(t, db, "SELECT COUNT(*) FROM players"), "player must not be created")
				assertMatchmakingIntact(t, db)
			},
		},
		"Delete player after matchmaking is rejected": {
			method:             "DELETE",
			endpoint:           "/games/1/teams/1/players/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Game setup already assigned, reset the setup first"}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()
				assert.Equal(t, 32, countRows(t, db, "SELECT COUNT(*) FROM players"), "player must not be deleted")
				assertMatchmakingIntact(t, db)
			},
		},
		"Delete player in a running game": {
			method:             "DELETE",
			endpoint:           "/games/1/teams/1/players/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Game is not editable"}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()
				assert.Equal(t, 32, countRows(t, db, "SELECT COUNT(*) FROM players"), "player must not be deleted")
				assertMatchmakingIntact(t, db)
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
