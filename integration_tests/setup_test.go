package integration_tests

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pressly/goose/v3"

	"github.com/henok321/knobel-manager-service/internal/app"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func readContentFromFile(t *testing.T, filepath string) string {
	content, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	return string(content)
}

func executeSQLFile(t *testing.T, db *sql.DB, filepath string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("failed to read SQL file: %v", err)
	}
	_, err = db.Exec(string(content))
	if err != nil {
		t.Fatalf("failed to execute SQL file: %v", err)
	}
	return nil
}

func runGooseUp(t *testing.T, db *sql.DB) error {
	migrationsDir := filepath.Join("..", "db_migration")
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("failed to set dialect: %v", err)
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
	return nil
}

func mockAuthMiddleware(sub string) gin.HandlerFunc {
	return func(c *gin.Context) {

		authorizationHeader := c.GetHeader("Authorization")
		if authorizationHeader != "sub-1" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		userInfo := map[string]interface{}{
			"sub":   sub,
			"email": "mock@example.com",
		}

		c.Set("user", userInfo)

		c.Next()
	}
}

func setupTestServer() (*httptest.Server, func(*httptest.Server)) {
	testInstance := &app.App{}
	testInstance.Initialize(mockAuthMiddleware("sub-1"))
	server := httptest.NewServer(testInstance.Router)
	teardown := func(*httptest.Server) {
		server.Close()
	}

	return server, teardown
}

func setupTestDatabase(t *testing.T) (string, func(), error) {
	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx, "docker.io/postgres:16-alpine", postgres.WithDatabase("knobel-manager-service"), postgres.WithUsername("test"), postgres.WithPassword("secret"), testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
		WithOccurrence(2).WithStartupTimeout(5*time.Second)))
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	if err := os.Setenv("DATABASE_URL", connStr); err != nil {
		t.Fatalf("failed to set DATABASE_URL: %v", err)
	}

	teardown := func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}

	return connStr, teardown, nil
}
