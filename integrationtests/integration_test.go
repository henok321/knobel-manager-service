package integrationtests

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
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
	"github.com/henok321/knobel-manager-service/pkg/entity"
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

	if err := goose.Up(db, migrationsDir); err != nil && !errors.Is(err, goose.ErrNoNextVersion) {
		t.Fatalf("goose failed to run migrations: %v", err)
	}
}

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	url := os.Getenv("DATABASE_URL")
	database, err := audit.OpenDatabase(url)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Registered before the server so it runs after it: Postgres has a finite
	// max_connections and this helper is called once per test.
	t.Cleanup(func() {
		if pool, err := database.DB(); err == nil {
			_ = pool.Close()
		}
	})

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
	t.Cleanup(server.Close)

	// Cleanups run after every defer, so a test that defers fixture teardown tears it down
	// with the server still up: await in-flight requests before returning.
	return server
}

var (
	sharedDatabase     *postgres.PostgresContainer
	sharedDatabaseURL  string
	errSharedDatabase  error
	sharedDatabaseOnce sync.Once
)

func TestMain(m *testing.M) {
	code := m.Run()

	if sharedDatabase != nil {
		if err := sharedDatabase.Terminate(context.Background()); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}

	os.Exit(code)
}

func disposableTestDatabase(t *testing.T) (string, func()) {
	t.Helper()

	ctx := context.Background()

	container, err := startPostgres(ctx)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	t.Setenv("DATABASE_URL", connStr)

	return connStr, func() {
		if err := container.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}
}

func startPostgres(ctx context.Context) (*postgres.PostgresContainer, error) {
	return postgres.Run(ctx, "docker.io/postgres:18-alpine",
		postgres.WithDatabase("knobel-manager-service"),
		postgres.WithUsername("test"),
		postgres.WithPassword("secret"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(5*time.Second)))
}

func setupTestDatabase(t *testing.T) string {
	t.Helper()

	sharedDatabaseOnce.Do(func() {
		ctx := context.Background()

		sharedDatabase, errSharedDatabase = startPostgres(ctx)
		if errSharedDatabase != nil {
			return
		}

		sharedDatabaseURL, errSharedDatabase = sharedDatabase.ConnectionString(ctx, "sslmode=disable")
	})

	if errSharedDatabase != nil {
		t.Fatalf("failed to start container: %v", errSharedDatabase)
	}

	t.Setenv("DATABASE_URL", sharedDatabaseURL)

	return sharedDatabaseURL
}

func advanceSequences(t *testing.T, db *sql.DB) {
	t.Helper()

	const query = `SELECT setval('teams_id_seq', COALESCE((SELECT MAX(id) FROM teams), 0) + 1, false),
		setval('players_id_seq', COALESCE((SELECT MAX(id) FROM players), 0) + 1, false)`

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

const (
	roundsOfGameOne = "SELECT COUNT(*) FROM rounds WHERE game_id = 1"
	tablesOfGameOne = "SELECT COUNT(*) FROM game_tables gt JOIN rounds r ON r.id = gt.round_id WHERE r.game_id = 1"
	seatsOfGameOne  = `SELECT COUNT(*) FROM table_players tp
		JOIN game_tables gt ON gt.id = tp.game_table_id
		JOIN rounds r ON r.id = gt.round_id WHERE r.game_id = 1`
)

func assertNoTablesAssigned(t *testing.T, db *sql.DB) {
	t.Helper()

	assert.Zero(t, countRows(t, db, roundsOfGameOne), "no round may survive")
	assert.Zero(t, countRows(t, db, tablesOfGameOne), "no table may survive")
	assert.Zero(t, countRows(t, db, seatsOfGameOne), "no seat may survive")
	assert.Zero(t, countRows(t, db, "SELECT COUNT(*) FROM scores"), "scores go with the tables they belong to")
}

func assertTablesAssigned(t *testing.T, db *sql.DB) {
	t.Helper()

	assert.NotZero(t, countRows(t, db, roundsOfGameOne), "the rounds must be untouched")
	assert.NotZero(t, countRows(t, db, tablesOfGameOne), "the tables must be untouched")
	assert.NotZero(t, countRows(t, db, seatsOfGameOne), "the seats must be untouched")
}

func gameStatus(t *testing.T, db *sql.DB) entity.GameStatus {
	t.Helper()

	var status entity.GameStatus
	if err := db.QueryRowContext(t.Context(), "SELECT status FROM games WHERE id = 1").Scan(&status); err != nil {
		t.Fatalf("failed to query game status: %v", err)
	}

	return status
}
