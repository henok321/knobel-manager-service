package integration_tests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/henok321/knobel-manager-service/pkg/game"

	"github.com/henok321/knobel-manager-service/internal/app"
	"github.com/stretchr/testify/assert"
)

func setup() (*app.App, *httptest.Server) {

	var testInstance app.App
	testInstance.InitializeTest()

	server := httptest.NewServer(testInstance.Router)
	return &testInstance, server
}

func teardown(server *httptest.Server) {
	server.Close()
}

func TestAppInitialization(t *testing.T) {

	t.Run("health check", func(t *testing.T) {
		_, server := setup()
		defer teardown(server)

		resp, err := http.Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to perform GET request: %v", err)
		}
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")
	})

	t.Run("get games empty", func(t *testing.T) {
		_, server := setup()
		defer teardown(server)

		resp, err := http.Get(server.URL + "/games")
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
		instance, server := setup()
		defer teardown(server)

		var owners1 = []game.Owner{
			{Sub: "mock-sub"},
		}

		var owners2 = []game.Owner{
			{Sub: "unknown-sub"},
		}

		instance.DB.Create(&game.Game{
			Name: "test game", Owners: owners1,
		})
		instance.DB.Create(&game.Game{
			Name: "test game 2", Owners: owners2,
		})

		resp, err := http.Get(server.URL + "/games")
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
