package integration_tests

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pressly/goose/v3"

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

func runGooseUp(db *sql.DB) error {
	migrationsDir := filepath.Join("..", "db_migration")
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

func mockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		authorizationHeader := c.GetHeader("Authorization")
		if authorizationHeader != "permitted" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		userInfo := map[string]interface{}{
			"sub":   "sub-1",
			"email": "mock@example.com",
		}

		c.Set("user", userInfo)

		c.Next()
	}
}

func setupTestServer() (*httptest.Server, func(*httptest.Server)) {
	testInstance := &app.App{}
	testInstance.Initialize(mockAuthMiddleware())
	server := httptest.NewServer(testInstance.Router)
	teardown := func(*httptest.Server) {
		server.Close()
	}

	return server, teardown
}

func setupTestDatabase() (string, func(), error) {
	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx, "docker.io/postgres:16-alpine",
		postgres.WithDatabase("knobel-manager-service"),
		postgres.WithUsername("test"),
		postgres.WithPassword("secret"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(5*time.Second)))
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to start container: %w", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to get connection string: %w", err)
	}

	if err := os.Setenv("DATABASE_URL", connStr); err != nil {
		return "", func() {}, fmt.Errorf("failed to set DATABASE_URL: %w", err)
	}

	teardown := func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}

	return connStr, teardown, nil
}
