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

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	healthpkg "github.com/henok321/knobel-manager-service/api/health"
	"github.com/henok321/knobel-manager-service/api/routes"
	"github.com/henok321/knobel-manager-service/integrationtests/mock"
	"github.com/henok321/knobel-manager-service/pkg/audit"
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
	assertions         func(t *testing.T, db *sql.DB)
}

func newTestRequest(t *testing.T, tc testCase, server *httptest.Server, db *sql.DB) {
	t.Helper()

	var requestBody io.Reader
	if tc.requestBody != "" {
		requestBody = bytes.NewBuffer([]byte(tc.requestBody))
	}

	req, err := http.NewRequestWithContext(t.Context(), tc.method, server.URL+tc.endpoint, requestBody)
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

	if tc.expectedHeaders != nil {
		for key, value := range tc.expectedHeaders {
			assert.Equal(t, value, resp.Header.Get(key))
		}
	}

	if tc.assertions != nil {
		tc.assertions(t, db)
	}
}

func readContentFromFile(t *testing.T, filepath string) string {
	t.Helper()

	content, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	return string(content)
}

func executeSQLFile(t *testing.T, db *sql.DB, filepath string) {
	t.Helper()

	content, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("failed to read SQL file: %v", err)
	}

	_, err = db.ExecContext(t.Context(), string(content))
	if err != nil {
		t.Fatalf("failed to execute SQL file: %v", err)
	}
}

func runGooseUp(t *testing.T, db *sql.DB) {
	t.Helper()

	migrationsDir := filepath.Join("..", "db_migration")

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose failed to set dialect: %v", err)
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		t.Fatalf("goose failed to run migrations: %v", err)
	}
}

func setupTestServer(t *testing.T) (*httptest.Server, func(*httptest.Server)) {
	t.Helper()

	url := os.Getenv("DATABASE_URL")
	database, err := audit.OpenDatabase(url)
	if err != nil {
		log.Fatalln("Starting application failed, cannot start connect to database", err)
	}

	dbChecker := healthpkg.NewDatabaseChecker(database, 500*time.Millisecond)
	firebaseChecker := healthpkg.NewFirebaseChecker(mock.FirebaseAuthMock{}, 500*time.Millisecond)
	healthService := healthpkg.NewService(dbChecker, firebaseChecker)

	openAPIConfig, err := os.ReadFile(filepath.Join("..", "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatal("Could not read openapi.yaml", err)
	}
	swaggerDocs, err := os.ReadFile(filepath.Join("..", "openapi", "swagger.html"))
	if err != nil {
		t.Fatal("Could not read swagger.html", err)
	}

	router := routes.SetupRouter(database, mock.FirebaseAuthMock{}, healthService, openAPIConfig, swaggerDocs)

	server := httptest.NewServer(router)
	teardown := func(*httptest.Server) {
		server.Close()
	}

	return server, teardown
}

func setupTestDatabase(t *testing.T) (string, func()) {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "docker.io/postgres:18-alpine", postgres.WithDatabase("knobel-manager-service"), postgres.WithUsername("test"), postgres.WithPassword("secret"), testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
		WithOccurrence(2).WithStartupTimeout(5*time.Second)))
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	t.Setenv("DATABASE_URL", connStr)

	teardown := func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}

	return connStr, teardown
}

func advanceSequences(t *testing.T, db *sql.DB) {
	t.Helper()

	const query = `SELECT setval('teams_id_seq', (SELECT COALESCE(MAX(id), 1) FROM teams)),
		setval('players_id_seq', (SELECT COALESCE(MAX(id), 1) FROM players))`

	if _, err := db.ExecContext(t.Context(), query); err != nil {
		t.Fatalf("failed to advance sequences: %v", err)
	}
}

func countRows(t *testing.T, db *sql.DB, query string) int {
	t.Helper()

	var count int
	if err := db.QueryRowContext(t.Context(), query).Scan(&count); err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}

	return count
}

func assertMatchmakingReset(t *testing.T, db *sql.DB) {
	t.Helper()

	assert.Zero(t, countRows(t, db, "SELECT COUNT(*) FROM rounds WHERE game_id = 1"), "rounds must be reset")
	assert.Zero(t, countRows(t, db, "SELECT COUNT(*) FROM game_tables"), "game tables must be reset")
	assert.Zero(t, countRows(t, db, "SELECT COUNT(*) FROM table_players"), "table assignments must be reset")
}

func assertMatchmakingIntact(t *testing.T, db *sql.DB) {
	t.Helper()

	assert.NotZero(t, countRows(t, db, "SELECT COUNT(*) FROM rounds WHERE game_id = 1"), "rounds must be untouched")
	assert.NotZero(t, countRows(t, db, "SELECT COUNT(*) FROM table_players"), "table assignments must be untouched")
}
