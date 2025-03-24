package integrationtests

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/henok321/knobel-manager-service/integrationtests/mock"
	pg "gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/internal/routes"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	method             string
	endpoint           string
	requestBody        string
	requestHeaders     map[string]string
	setup              func(db *sql.DB)
	expectedStatusCode int
	expectedBody       string
	expectedHeaders    map[string]string
}

func newTestRequest(t *testing.T, tc testCase, server *httptest.Server) {
	var requestBody io.Reader
	if tc.requestBody != "" {
		requestBody = bytes.NewBuffer([]byte(tc.requestBody))
	}

	// Create the HTTP request
	req, err := http.NewRequest(tc.method, server.URL+tc.endpoint, requestBody)
	if err != nil {
		t.Fatalf("Failed to create %s request: %v", tc.method, err)
	}

	for key, value := range tc.requestHeaders {
		req.Header.Set(key, value)
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to perform %s request: %v", tc.method, err)
	}
	defer resp.Body.Close()

	responseBodyBytes, _ := io.ReadAll(resp.Body)
	responseBodyString := string(responseBodyBytes)

	assert.Equal(t, tc.expectedStatusCode, resp.StatusCode, "Expected status code %d", tc.expectedStatusCode)

	if tc.expectedBody != "" {
		assert.JSONEq(t, tc.expectedBody, responseBodyString)
	}
}

func readContentFromFile(t *testing.T, filepath string) string {
	content, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	return string(content)
}

func executeSQLFile(t *testing.T, db *sql.DB, filepath string) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("failed to read SQL file: %v", err)
	}

	_, err = db.Exec(string(content))
	if err != nil {
		t.Fatalf("failed to execute SQL file: %v", err)
	}
}

func runGooseUp(t *testing.T, db *sql.DB) {
	migrationsDir := filepath.Join("..", "db_migration")

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose failed to set dialect: %v", err)
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		t.Fatalf("goose failed to run migrations: %v", err)
	}
}

func setupTestServer() (*httptest.Server, func(*httptest.Server)) {
	url := os.Getenv("DATABASE_URL")
	database, err := gorm.Open(pg.Open(url), &gorm.Config{})
	if err != nil {
		log.Fatalln("Starting application failed, cannot start connect to database", err)
	}

	router := routes.SetupRouter(database, mock.FirebaseAuthMock{})

	server := httptest.NewServer(router)
	teardown := func(*httptest.Server) {
		server.Close()
	}

	return server, teardown
}

func setupTestDatabase(t *testing.T) (string, func()) {
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

	return connStr, teardown
}
