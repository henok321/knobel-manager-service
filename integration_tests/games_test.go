package integration_tests

import (
	"bytes"
	"database/sql"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGames(t *testing.T) {
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

	tests := []testCase{
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

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {
			cleanup, _ := setupTestDatabase()
			defer cleanup()

			server, teardown, db := setupTestServer()
			defer teardown(server)
			defer db.Close()

			tc.setup(db)

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
