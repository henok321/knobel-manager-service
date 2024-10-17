package integration_tests

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

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

func startTestDatabase() (*sql.DB, func(), error) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "docker.io/postgres:16-alpine", postgres.WithInitScripts(filepath.Join("..", "testdata", "init-db.sql")), // Path to your init script
		postgres.WithDatabase("knobel-manager-service"), postgres.WithUsername("test"), postgres.WithPassword("secret"), testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(5*time.Second)))
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to start container: %w", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to get connection string: %w", err)
	}

	if err := os.Setenv("DATABASE_URL", connStr); err != nil {
		return nil, func() {}, fmt.Errorf("failed to set DATABASE_URL: %w", err)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to open database: %w", err)
	}

	cleanup := func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
		db.Close()
	}

	// Return the database and cleanup function
	return db, cleanup, nil
}

func TestAppInitialization(t *testing.T) {

	t.Run("health check", func(t *testing.T) {
		_, cleanup, _ := startTestDatabase()
		defer cleanup()

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
		_, cleanup, _ := startTestDatabase()
		defer cleanup()

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
		_, cleanup, _ := startTestDatabase()
		defer cleanup()

		instance, server := setup()
		defer teardown(server)

		owners1 := []game.Owner{
			{Sub: "mock-sub"},
		}

		owners2 := []game.Owner{
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
