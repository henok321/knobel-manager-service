package integrationtests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/lib/pq"
)

func TestScores(t *testing.T) {
	tests := map[string]testCase{
		"Add new score for game": {
			method:             "PUT",
			endpoint:           "/games/1/rounds/1/tables/1/scores",
			expectedStatusCode: http.StatusOK,
			requestBody:        `{"scores": [{"playerID":1,"score":6},{"playerID":5,"score":3},{"playerID":9,"score":2},{"playerID":13,"score":1}]}`,
			expectedBody:       readContentFromFile(t, "./integrationtests/test_data/game_update_score_response.json"),
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./integrationtests/test_data/games_setup_assigned.sql")
			},
		},
		"Update existing score for game": {
			method:             "PUT",
			endpoint:           "/games/1/rounds/1/tables/1/scores",
			expectedStatusCode: http.StatusOK,
			requestBody:        `{"scores": [{"playerID":1,"score":6},{"playerID":5,"score":3},{"playerID":9,"score":2},{"playerID":13,"score":1}]}`,
			expectedBody:       readContentFromFile(t, "./integrationtests/test_data/game_update_score_response.json"),
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./integrationtests/test_data/games_setup_assigned_with_scores.sql")
			},
		},
		"Update score not game owner": {
			method:             "PUT",
			endpoint:           "/games/1/rounds/1/tables/1/scores",
			expectedStatusCode: http.StatusNotFound,
			requestBody:        `{"scores": [{"playerID":1,"score":6},{"playerID":5,"score":3},{"playerID":9,"score":2},{"playerID":13,"score":1}]}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./integrationtests/test_data/games_setup_assigned.sql")
			},
		},
		"Update score game not found": {
			method:             "PUT",
			endpoint:           "/games/2/rounds/1/tables/1/scores",
			expectedStatusCode: http.StatusNotFound,
			requestBody:        `{"scores": [{"playerID":1,"score":6},{"playerID":5,"score":3},{"playerID":9,"score":2},{"playerID":13,"score":1}]}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./integrationtests/test_data/games_setup_assigned.sql")
			},
		},
		"Update score round not found": {
			method:             "PUT",
			endpoint:           "/games/1/rounds/2/tables/1/scores",
			expectedStatusCode: http.StatusNotFound,
			requestBody:        `{"scores": [{"playerID":1,"score":6},{"playerID":5,"score":3},{"playerID":9,"score":2},{"playerID":13,"score":1}]}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./integrationtests/test_data/games_setup_assigned.sql")
			},
		},
		"Update score table not found": {
			method:             "PUT",
			endpoint:           "/games/1/rounds/1/tables/35/scores",
			expectedStatusCode: http.StatusNotFound,
			requestBody:        `{"scores": [{"playerID":1,"score":6},{"playerID":5,"score":3},{"playerID":9,"score":2},{"playerID":13,"score":1}]}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./integrationtests/test_data/games_setup_assigned.sql")
			},
		},
		"Update score invalid gameID": {
			method:             "PUT",
			endpoint:           "/games/invalid/rounds/1/tables/1/scores",
			expectedStatusCode: http.StatusBadRequest,
			requestBody:        `{"scores": [{"playerID":1,"score":6},{"playerID":5,"score":3},{"playerID":9,"score":2},{"playerID":13,"score":1}]}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./integrationtests/test_data/games_setup_assigned.sql")
			},
		},
		"Update score invalid roundNumber": {
			method:             "PUT",
			endpoint:           "/games/1/rounds/invalid/tables/1/scores",
			expectedStatusCode: http.StatusBadRequest,
			requestBody:        `{"scores": [{"playerID":1,"score":6},{"playerID":5,"score":3},{"playerID":9,"score":2},{"playerID":13,"score":1}]}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./integrationtests/test_data/games_setup_assigned.sql")
			},
		},
		"Update score invalid tableNumber": {
			method:             "PUT",
			endpoint:           "/games/1/rounds/1/tables/invalid/scores",
			expectedStatusCode: http.StatusBadRequest,
			requestBody:        `{"scores": [{"playerID":1,"score":6},{"playerID":5,"score":3},{"playerID":9,"score":2},{"playerID":13,"score":1}]}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./integrationtests/test_data/games_setup_assigned.sql")
			},
		},
		"Update score invalid request body": {
			method:             "PUT",
			endpoint:           "/games/1/rounds/1/tables/1/scores",
			expectedStatusCode: http.StatusBadRequest,
			requestBody:        `{"scores": [{"playerID":1,"score":"invalid"},{"playerID":5,"score":3},{"playerID":9,"score":2},{"playerID":13,"score":1}]}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./integrationtests/test_data/games_setup_assigned.sql")
			},
		},
	}

	dbConn, teardownDatabase := setupTestDatabase(t)
	defer teardownDatabase()

	db, err := sql.Open("postgres", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			t.Fatalf("Failed to close database connection: %v", err)
		}
	}(db)

	runGooseUp(t, db)

	server, teardown := setupTestServer()
	defer teardown(server)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(db)
			}

			defer executeSQLFile(t, db, "./integrationtests/test_data/cleanup.sql")
			newTestRequest(t, tc, server, db)
		})
	}
}
