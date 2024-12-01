package integration_tests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/lib/pq"
)

func TestScores(t *testing.T) {
	tests := map[string]testCase{
		"Update score for game": {
			method:             "PUT",
			endpoint:           "/games/1/rounds/1/tables/1/scores",
			expectedStatusCode: http.StatusOK,
			requestBody:        `{"scores": []}`,
			expectedBody:       `{"game": {}}`,
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
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
