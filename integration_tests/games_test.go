package integration_tests

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/henok321/knobel-manager-service/pkg/game"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestGames(t *testing.T) {
	type testCase struct {
		name               string
		method             string
		endpoint           string
		body               string
		setup              func(db *gorm.DB)
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
			setup: func(db *gorm.DB) {
				// No setup needed; database is empty
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"games":[]}`,
			headers:            map[string]string{"Authorization": "permitted"},
		},
		// GET /games with data
		{
			name:     "GET games by data",
			method:   "GET",
			endpoint: "/games",
			setup: func(db *gorm.DB) {
				owners1 := []game.Owner{{Sub: "mock-sub"}}
				owners2 := []game.Owner{{Sub: "unknown-sub"}}

				db.Create(&game.Game{Name: "test game", Owners: owners1})
				db.Create(&game.Game{Name: "test game 2", Owners: owners2})
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"games":[{"id":1,"name":"test game","owners":[{"id":1,"sub":"mock-sub"}]}]}`,
			headers:            map[string]string{"Authorization": "permitted"},
		},
		// POST /games to create a new game
		{
			name:     "POST create game",
			method:   "POST",
			endpoint: "/games",
			body:     `{"name":"new game"}`,
			setup: func(db *gorm.DB) {
				// No initial setup needed
			},
			expectedStatusCode: http.StatusCreated,
			expectedBody:       `{"id":1,"name":"new game","owners":[{"id":1,"sub":"mock-sub"}]}`,
			headers:            map[string]string{"Authorization": "permitted", "Content-Type": "application/json"},
		},
		// PUT /games/:id to update a game
		{
			name:   "PUT update game",
			method: "PUT",
			// Assuming the game ID is 1
			endpoint: "/games/1",
			body:     `{"name":"updated game"}`,
			setup: func(db *gorm.DB) {
				owners := []game.Owner{{Sub: "mock-sub"}}
				db.Create(&game.Game{ID: 1, Name: "existing game", Owners: owners})
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"id":1,"name":"updated game","owners":[{"id":1,"sub":"mock-sub"}]}`,
			headers:            map[string]string{"Authorization": "permitted", "Content-Type": "application/json"},
		},
		// DELETE /games/:id to delete a game
		{
			name:     "DELETE game",
			method:   "DELETE",
			endpoint: "/games/1",
			setup: func(db *gorm.DB) {
				owners := []game.Owner{{Sub: "mock-sub"}}
				db.Create(&game.Game{ID: 1, Name: "game to delete", Owners: owners})
			},
			expectedStatusCode: http.StatusNoContent,
			expectedBody:       ``,
			headers:            map[string]string{"Authorization": "permitted"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, cleanup, _ := setupTestDatabase()
			defer cleanup()

			instance, server, teardown := setupTestServer()
			defer teardown(server)

			// Perform the setup specific to this test case
			tc.setup(instance.DB)

			// Prepare the request body
			var requestBody io.Reader
			if tc.body != "" {
				requestBody = bytes.NewBuffer([]byte(tc.body))
			}

			// Create the HTTP request
			req, err := http.NewRequest(tc.method, server.URL+tc.endpoint, requestBody)
			if err != nil {
				t.Fatalf("Failed to create %s request: %v", tc.method, err)
			}

			// Set headers
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			// Execute the HTTP request
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to perform %s request: %v", tc.method, err)
			}
			defer resp.Body.Close()

			// Read the response body
			bodyBytes, _ := io.ReadAll(resp.Body)
			bodyString := string(bodyBytes)

			// Assertions
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
