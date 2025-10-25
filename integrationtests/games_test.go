package integrationtests

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/henok321/knobel-manager-service/pkg/entity"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestGames(t *testing.T) {
	tests := map[string]testCase{
		"Find games empty": {
			method:             http.MethodGet,
			endpoint:           "/games",
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"games":[]}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
		},
		"Find games": {
			method:   http.MethodGet,
			endpoint: "/games",
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			}, expectedStatusCode: http.StatusOK,
			expectedBody:   readContentFromFile(t, "./test_data/games_setup.json"),
			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
		},
		"Find game by id ok": {
			method:   http.MethodGet,
			endpoint: "/games/1",
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			}, expectedStatusCode: http.StatusOK,
			expectedBody:   readContentFromFile(t, "./test_data/games_setup_by_id.json"),
			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
		},
		"Find game by id not found": {
			method:   http.MethodGet,
			endpoint: "/games/2",
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			}, expectedStatusCode: http.StatusNotFound,
			expectedBody:   `{"error":"Game not found"}`,
			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
		},
		"Find game by id Invalid gameID": {
			method:   http.MethodGet,
			endpoint: "/games/invalid",
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			}, expectedStatusCode: http.StatusBadRequest,
			expectedBody:   `{"error":"Invalid gameID"}`,
			requestHeaders: map[string]string{"Authorization": "Bearer sub-1"},
		},
		"Find game by id not owner": {
			method:   http.MethodGet,
			endpoint: "/games/1",
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
			expectedStatusCode: http.StatusForbidden,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
		},
		"Create new game": {
			method:             http.MethodPost,
			endpoint:           "/games",
			expectedStatusCode: http.StatusCreated,
			requestBody:        `{"name":"Game 1","numberOfRounds":2, "teamSize":4, "tableSize":4}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedBody:       `{"game":{"id":1,"name":"Game 1","teamSize":4,"tableSize":4,"numberOfRounds":2,"status":"setup","owners":[{"gameID":1,"ownerSub":"sub-1"}]}}`,
			expectedHeaders:    map[string]string{"Location": "/games/1"},
		},
		"Create new game invalid request": {
			method:             http.MethodPost,
			endpoint:           "/games",
			expectedStatusCode: http.StatusBadRequest,
			requestBody:        `{}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
		},
		"Update an existing game": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1 updated","numberOfRounds":3, "teamSize":4, "tableSize":4}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"game":{"id":1,"name":"Game 1 updated","teamSize":4,"tableSize":4,"numberOfRounds":3,"status":"setup","owners":[{"gameID":1,"ownerSub":"sub-1"}]}}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Update game status to in_progress": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1","numberOfRounds":2, "teamSize":4, "tableSize":4, "status":"in_progress"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"game":{"id":1,"name":"Game 1","teamSize":4,"tableSize":4,"numberOfRounds":2,"status":"in_progress","owners":[{"gameID":1,"ownerSub":"sub-1"}]}}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assignable.sql")
			},
		},
		"Update game status to in_progress with invalid setup": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1","numberOfRounds":2, "teamSize":4, "tableSize":4, "status":"in_progress"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_not_assignable.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				var gameStatus *string

				if err := db.QueryRow("SELECT status FROM games WHERE id = 1").Scan(&gameStatus); err != nil {
					t.Fatalf("Failed to query game status: %v", err)
				}

				assert.Equal(t, entity.StatusSetup, entity.GameStatus(*gameStatus))
			},
		},
		"Update an existing game invalid request": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1 updated","numberOfRounds":3}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusBadRequest,
		},
		"Update an existing Game not found": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1 updated","numberOfRounds":3, "teamSize":4, "tableSize":4}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
		},
		"Update an existing game not owner": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1 updated","numberOfRounds":3, "teamSize":4, "tableSize":4}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Delete an existing game": {
			method:             http.MethodDelete,
			endpoint:           "/games/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Delete an existing Game not found": {
			method:             http.MethodDelete,
			endpoint:           "/games/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
		},
		"Delete an existing game not owner": {
			method:             http.MethodDelete,
			endpoint:           "/games/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Set active game": {
			method:             http.MethodPost,
			endpoint:           "/games/1/activate",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Set active game forbidden": {
			method:             http.MethodPost,
			endpoint:           "/games/1/activate",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Set active game not found": {
			method:             http.MethodPost,
			endpoint:           "/games/2/activate",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Switch active game": {
			method:             http.MethodPost,
			endpoint:           "/games/2/activate",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_switch_active.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				var activeGameID int
				err := db.QueryRow("SELECT game_id FROM active_games WHERE owner_sub = 'sub-1'").Scan(&activeGameID)
				if err != nil {
					t.Fatalf("Failed to query active game: %v", err)
				}
				if activeGameID != 2 {
					t.Errorf("Expected active game to be 2, got %d", activeGameID)
				}

				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM active_games WHERE owner_sub = 'sub-1'").Scan(&count)
				if err != nil {
					t.Fatalf("Failed to count active games: %v", err)
				}
				if count != 1 {
					t.Errorf("Expected only 1 active game, got %d", count)
				}
			},
		},
		"Verify active game after switch": {
			method:             http.MethodGet,
			endpoint:           "/games",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			expectedBody:       readContentFromFile(t, "./test_data/games_switch_active.json"),
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_switch_active.sql")
				_, err := db.Exec("DELETE FROM active_games WHERE owner_sub = 'sub-1'")
				if err != nil {
					t.Fatalf("Failed to delete active games: %v", err)
				}
				_, err = db.Exec("INSERT INTO active_games (game_id, owner_sub) VALUES (2, 'sub-1')")
				if err != nil {
					t.Fatalf("Failed to insert active game: %v", err)
				}
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
