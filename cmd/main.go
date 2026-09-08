package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/api/option"
	"gorm.io/gorm"

	healthpkg "github.com/henok321/knobel-manager-service/api/health"
	"github.com/henok321/knobel-manager-service/api/logging"
	"github.com/henok321/knobel-manager-service/api/routes"
	"github.com/henok321/knobel-manager-service/pkg/audit"
)

func init() {
	switch os.Getenv("ENVIRONMENT") {
	case "local":
		logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: false, Level: slog.LevelDebug})
		slog.SetDefault(slog.New(&logging.ContextHandler{Handler: logHandler}))
		slog.Info("Logging initialized", "logLevel", "debug")
	default:
		logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: false, Level: slog.LevelInfo})
		slog.SetDefault(slog.New(&logging.ContextHandler{Handler: logHandler}))
		slog.Info("Logging initialized", "logLevel", "info")
	}
}

func setupAuthClient() (*auth.Client, error) {
	encoded := strings.NewReplacer("=", "", "\n", "", "\r", "").Replace(os.Getenv("FIREBASE_SECRET"))

	firebaseSecret, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	if len(firebaseSecret) == 0 {
		return nil, errors.New("FIREBASE_SECRET is undefined or empty")
	}

	firebaseOption := option.WithAuthCredentialsJSON(option.ServiceAccount, firebaseSecret)
	firebaseApp, err := firebase.NewApp(context.Background(), nil, firebaseOption)
	if err != nil {
		return nil, err
	}

	return firebaseApp.Auth(context.Background())
}

func setupDatabase() (*gorm.DB, *sql.DB, error) {
	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		return nil, nil, errors.New("DATABASE_URL is not set")
	}

	gormDB, err := audit.OpenDatabase(databaseURL)
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, nil, err
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	return gormDB, sqlDB, nil
}

func runDatabaseMigrations(db *sql.DB) error {
	slog.Info("Running database migrations")

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	migrationsDir := os.Getenv("DB_MIGRATION_DIR")
	if migrationsDir == "" {
		return errors.New("migrations directory is not set")
	}

	migrationsDir = filepath.Clean(migrationsDir)

	slog.Info("Using migrations directory", "path", migrationsDir)

	if err := goose.Up(db, migrationsDir); err != nil {
		return err
	}

	slog.Info("Database migrations completed successfully")
	return nil
}

func setupOpenAPIConfig() (openAPIConfig, swaggerDocs []byte, err error) {
	openAPIConfig, err = os.ReadFile(filepath.Join("openapi", "openapi.yaml"))
	if err != nil {
		return nil, nil, err
	}

	swaggerDocs, err = os.ReadFile(filepath.Join("openapi", "swagger.html"))
	if err != nil {
		return nil, nil, err
	}

	return openAPIConfig, swaggerDocs, nil
}

// Echoes the request origin instead of "*": browsers reject
// Access-Control-Allow-Origin: * when credentials are allowed.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "300")
			w.WriteHeader(http.StatusNoContent)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	slog.Info("Initialize application")

	authClient, err := setupAuthClient()
	if err != nil {
		slog.Error("Failed to initialize auth client", "error", err)
		os.Exit(1)
	}

	gormDB, sqlDB, err := setupDatabase()
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	if err := runDatabaseMigrations(sqlDB); err != nil {
		slog.Error("Database migrations failed", "error", err)
		os.Exit(1)
	}

	dbChecker := healthpkg.NewDatabaseChecker(gormDB, 500*time.Millisecond)
	firebaseChecker := healthpkg.NewFirebaseChecker(authClient, 500*time.Millisecond)
	healthService := healthpkg.NewService(dbChecker, firebaseChecker)

	openAPIConfig, swaggerDocs, err := setupOpenAPIConfig()
	if err != nil {
		slog.Error("Failed to load OpenAPI config", "error", err)
		os.Exit(1)
	}

	router := routes.SetupRouter(gormDB, authClient, healthService, openAPIConfig, swaggerDocs)

	mainServer := &http.Server{
		Addr:         ":8080",
		Handler:      corsMiddleware(router),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	metricsRouter := http.NewServeMux()
	metricsRouter.Handle("GET /metrics", promhttp.Handler())

	metricsServer := &http.Server{
		Addr:         ":9090",
		Handler:      metricsRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("Starting main server", "port", 8080)

		if err := mainServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Main server error", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		slog.Info("Starting metrics server", "port", 9090)

		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Metrics server error", "error", err)
			os.Exit(1)
		}
	}()

	<-signalCtx.Done()
	slog.Info("Shutdown signal received, shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := mainServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Main server shutdown failed", "error", err)
	}

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Metrics server shutdown failed", "error", err)
	}

	slog.Info("Servers exited")
}
