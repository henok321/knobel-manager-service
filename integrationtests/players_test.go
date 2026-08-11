package integrationtests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
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

				var count int
				err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM players WHERE id = 1").Scan(&count)
				if err != nil {
					t.Fatalf("failed to count players: %v", err)
				}
				if count != 1 {
					t.Errorf("player must not be deleted by a non-owner, got count %d", count)
				}
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
