package integration_tests

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/henok321/knobel-manager-service/internal/app"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func executeSQLFile(db *sql.DB, filepath string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}
	_, err = db.Exec(string(content))
	if err != nil {
		return fmt.Errorf("failed to execute SQL file: %w", err)
	}
	return nil
}

func setupTestServer() (*httptest.Server, func(*httptest.Server), *sql.DB) {
	var testInstance app.App
	testInstance.InitializeTest()
	server := httptest.NewServer(testInstance.Router)
	teardown := func(*httptest.Server) {
		server.Close()
	}

	db, _ := testInstance.DB.DB()

	return server, teardown, db
}

func setupTestDatabase() (func(), error) {
	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx, "docker.io/postgres:16-alpine",
		postgres.WithInitScripts(filepath.Join("..", "testdata", "init.sql")),
		postgres.WithDatabase("knobel-manager-service"),
		postgres.WithUsername("test"),
		postgres.WithPassword("secret"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(5*time.Second)))
	if err != nil {
		return func() {}, fmt.Errorf("failed to start container: %w", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return func() {}, fmt.Errorf("failed to get connection string: %w", err)
	}

	if err := os.Setenv("DATABASE_URL", connStr); err != nil {
		return func() {}, fmt.Errorf("failed to set DATABASE_URL: %w", err)
	}

	cleanup := func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}

	return cleanup, nil
}
