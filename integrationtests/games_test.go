package integrationtests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"

	"github.com/henok321/knobel-manager-service/pkg/entity"
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
		"Find game by id returns round skeletons without tables": {
			method:   http.MethodGet,
			endpoint: "/games/1",
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned_with_scores.sql")
			}, expectedStatusCode: http.StatusOK,
			expectedBody:   readContentFromFile(t, "./test_data/games_setup_by_id_tables_scores.json"),
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
			expectedBody:   `{"error":"Invalid format for parameter gameID: error binding string parameter: strconv.ParseInt: parsing \"invalid\": invalid syntax"}`,
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
			expectedBody:       `{"game":{"id":1,"name":"Game 1","teamSize":4,"tableSize":4,"numberOfRounds":2,"status":"setup","owners":[{"gameID":1,"ownerSub":"sub-1","email":"sub-1@example.org"}]}}`,
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
			expectedBody:       `{"game":{"id":1,"name":"Game 1 updated","teamSize":4,"tableSize":4,"numberOfRounds":3,"status":"setup","owners":[{"gameID":1,"ownerSub":"sub-1","email":"sub-1@example.org"}]}}`,
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
			expectedBody:       `{"game":{"id":1,"name":"Game 1","teamSize":4,"tableSize":4,"numberOfRounds":2,"status":"in_progress","owners":[{"gameID":1,"ownerSub":"sub-1","email":"sub-1@example.org"}]}}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Equal(t, entity.StatusInProgress, gameStatus(t, db))
			},
		},
		"Update game status to in_progress without setup": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1","numberOfRounds":2, "teamSize":4, "tableSize":4, "status":"in_progress"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assignable.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Equal(t, entity.StatusSetup, gameStatus(t, db))
			},
		},
		"Should fail to update game status to completed if scores incomplete": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1","numberOfRounds":1, "teamSize":4, "tableSize":4, "status":"completed"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Equal(t, entity.StatusInProgress, gameStatus(t, db))
			},
		},
		"Update game status to completed": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1","numberOfRounds":1, "teamSize":4, "tableSize":4, "status":"completed"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"game":{"id":1,"name":"Game 1","teamSize":4,"tableSize":4,"numberOfRounds":1,"status":"completed","owners":[{"gameID":1,"ownerSub":"sub-1","email":"sub-1@example.org"}]}}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned_scores_entered.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Equal(t, entity.StatusCompleted, gameStatus(t, db))
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
				t.Helper()

				assert.Equal(t, entity.StatusSetup, gameStatus(t, db))
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
		"Update game status back to setup while nothing is scored": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1","numberOfRounds":1, "teamSize":4, "tableSize":4, "status":"setup"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Equal(t, entity.StatusSetup, gameStatus(t, db))
			},
		},
		"Update game status back to setup once scores exist": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1","numberOfRounds":1, "teamSize":4, "tableSize":4, "status":"setup"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Invalid status transition"}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned_with_scores.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Equal(t, entity.StatusInProgress, gameStatus(t, db))
				assert.Equal(t, 4, countRows(t, db, "SELECT COUNT(*) FROM scores"), "scores must survive a rejected transition")
			},
		},
		"Update game status from completed back to in_progress": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1","numberOfRounds":1, "teamSize":4, "tableSize":4, "status":"in_progress"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Invalid status transition"}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned_scores_entered.sql")
				if _, err := db.ExecContext(t.Context(), "UPDATE games SET status = 'completed' WHERE id = 1"); err != nil {
					t.Fatalf("Failed to update game status: %v", err)
				}
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Equal(t, entity.StatusCompleted, gameStatus(t, db))
			},
		},
		"Update game status from completed back to setup": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1","numberOfRounds":1, "teamSize":4, "tableSize":4, "status":"setup"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Invalid status transition"}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned_scores_entered.sql")
				if _, err := db.ExecContext(t.Context(), "UPDATE games SET status = 'completed' WHERE id = 1"); err != nil {
					t.Fatalf("Failed to update game status: %v", err)
				}
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Equal(t, entity.StatusCompleted, gameStatus(t, db))
			},
		},
		"Update game status to an unknown value": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1","numberOfRounds":2, "teamSize":4, "tableSize":4, "status":"banana"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       `{"error":"Invalid status"}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Equal(t, entity.StatusSetup, gameStatus(t, db))
			},
		},
		"Change the table size once the tables are assigned": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1","numberOfRounds":2, "teamSize":4, "tableSize":2, "status":"setup"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Game setup already assigned, reset the setup first"}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				var tableSize int
				if err := db.QueryRowContext(t.Context(), "SELECT table_size FROM games WHERE id = 1").Scan(&tableSize); err != nil {
					t.Fatalf("Failed to query table size: %v", err)
				}

				assert.Equal(t, 4, tableSize, "the assignment must keep the size it was built for")
			},
		},
		"Rename a game once the tables are assigned": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1 renamed","numberOfRounds":2, "teamSize":4, "tableSize":4, "status":"setup"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
			assertions: assertMatchmakingIntact,
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
