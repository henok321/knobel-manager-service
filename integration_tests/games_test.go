package integration_tests

import (
	"io"
	"net/http"
	"testing"

	"github.com/henok321/knobel-manager-service/pkg/game"
	"github.com/stretchr/testify/assert"
)

func TestGames(t *testing.T) {

	t.Run("get games empty", func(t *testing.T) {
		_, cleanup, _ := setupTestDatabase()
		defer cleanup()

		_, server, teardown := setupTestServer()
		defer teardown(server)

		req, err := http.NewRequest("GET", server.URL+"/games", nil)

		if err != nil {
			t.Fatalf("Failed to create GET request: %v", err)
		}

		req.Header.Set("Authorization", "permitted")

		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			t.Fatalf("Failed to perform GET request: %v", err)
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")
		assert.JSONEq(t, `{"games":[]}`, bodyString)
	})

	t.Run("get games", func(t *testing.T) {
		_, cleanup, _ := setupTestDatabase()
		defer cleanup()

		instance, server, teardown := setupTestServer()
		defer teardown(server)

		owners1 := []game.Owner{{Sub: "mock-sub"}}
		owners2 := []game.Owner{{Sub: "unknown-sub"}}

		instance.DB.Create(&game.Game{Name: "test game", Owners: owners1})
		instance.DB.Create(&game.Game{Name: "test game 2", Owners: owners2})

		req, err := http.NewRequest("GET", server.URL+"/games", nil)

		if err != nil {
			t.Fatalf("Failed to create GET request: %v", err)
		}

		req.Header.Set("Authorization", "permitted")

		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			t.Fatalf("Failed to perform GET request: %v", err)
		}

		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")
		assert.JSONEq(t, `{"games":[{"id":1,"name":"test game","owners":[{"id":1,"sub":"mock-sub"}]}]}`, bodyString)
	})
}
