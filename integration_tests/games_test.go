package integration_tests

import (
	"bytes"
	"database/sql"
	"io"
	"log"
	"net/http"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name               string
	method             string
	endpoint           string
	body               string
	setup              func(db *sql.DB)
	expectedStatusCode int
	expectedBody       string
	headers            map[string]string
}

func testCases(t *testing.T) []testCase {
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
			headers:            map[string]string{"Authorization": "permitted"},
		}, // GET /games with data
		{
			name:     "GET games by data",
			method:   "GET",
			endpoint: "/games",
			setup: func(db *sql.DB) {
				err := executeSQLFile(db, "../testdata/games/games_get_by_owner.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"games":[{"id":1,"name":"game 1","owners":[{"id":1,"sub":"sub-1"}]}]}`,
			headers:            map[string]string{"Authorization": "permitted"},
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
			headers:            map[string]string{"Authorization": "permitted", "Content-Type": "application/json"},
		}, // PUT /games/:id to update a game
		{
			name:     "PUT update game",
			method:   "PUT", // Assuming the game ID is 1
			endpoint: "/games/1",
			body:     `{"name":"updated game"}`,
			setup: func(db *sql.DB) {
				err := executeSQLFile(db, "../testdata/games/games_update.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"id":1,"name":"updated game","owners":[{"id":1,"sub":"sub-1"}]}`,
			headers:            map[string]string{"Authorization": "permitted", "Content-Type": "application/json"},
		}, // DELETE /games/:id to delete a game
		{
			name:     "DELETE game",
			method:   "DELETE",
			endpoint: "/games/1",
			setup: func(db *sql.DB) {
				err := executeSQLFile(db, "../testdata/games/games_delete.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
			expectedStatusCode: http.StatusNoContent,
			expectedBody:       ``,
			headers:            map[string]string{"Authorization": "permitted"},
		},
	}
}

func cleanupSetup(db *sql.DB, filepath string) {
	err := executeSQLFile(db, filepath)
	if err != nil {
		log.Fatalf("Failed to execute SQL file: %v", err)
	}

}

func TestGames(t *testing.T) {

	tests := testCases(t)
	dbConn, teardownDatabase, _ := setupTestDatabase()
	defer teardownDatabase()
	db, err := sql.Open("postgres", dbConn)

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatalf("Failed to close database connection: %v", err)
		}
	}(db)

	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {

			server, teardown := setupTestServer()
			defer teardown(server)

			tc.setup(db)
			defer cleanupSetup(db, "../testdata/cleanup.sql")

			var requestBody io.Reader
			if tc.body != "" {
				requestBody = bytes.NewBuffer([]byte(tc.body))
			}

			// Create the HTTP request
			req, err := http.NewRequest(tc.method, server.URL+tc.endpoint, requestBody)
			if err != nil {
				t.Fatalf("Failed to create %s request: %v", tc.method, err)
			}

			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to perform %s request: %v", tc.method, err)
			}
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)
			bodyString := string(bodyBytes)

			assert.Equal(t, tc.expectedStatusCode, resp.StatusCode, "Expected status code %d", tc.expectedStatusCode)

			if tc.expectedBody != "" {
				assert.JSONEq(t, tc.expectedBody, bodyString)
			} else {
				// For responses with no body
				assert.Empty(t, bodyString)
			}
		})
	}
}