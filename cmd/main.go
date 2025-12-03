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
	"strconv"
	"syscall"
	"time"

	healthpkg "github.com/henok321/knobel-manager-service/api/health"
	"github.com/henok321/knobel-manager-service/api/logging"
	"github.com/rs/cors"

	firebase "firebase.google.com/go/v4"

	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"google.golang.org/api/option"

	"github.com/henok321/knobel-manager-service/internal/routes"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

func runDatabaseMigrations(db *sql.DB) error {
	slog.Info("Running database migrations")

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	migrationsDir := os.Getenv("DB_MIGRATION_DIR")
	if migrationsDir == "" {
		slog.Error("Migrations directory is not set")
		return errors.New("migrations directory is not set")
	}

	slog.Info("Using migrations directory", "path", migrationsDir)

	if err := goose.Up(db, migrationsDir); err != nil {
		return err
	}

	slog.Info("Database migrations completed successfully")
	return nil
}

func main() {
	exitCode := 0

	defer func() {
		os.Exit(exitCode)
	}()

	slog.Info("Initialize application")

	firebaseSecret, err := base64.RawStdEncoding.DecodeString(os.Getenv("FIREBASE_SECRET"))
	if err != nil {
		slog.Error("Starting application failed, cannot decode FIREBASE_SECRET", "error", err)
		exitCode = 1
		return
	}

	if len(firebaseSecret) == 0 {
		slog.Error("Starting application failed, FIREBASE_SECRET is undefined or empty")
		exitCode = 1
		return
	}

	firebaseOption := option.WithCredentialsJSON(firebaseSecret)
	firebaseApp, err := firebase.NewApp(context.Background(), nil, firebaseOption)
	if err != nil {
		slog.Error("Starting application failed, cannot initialize firebase client. Check if the environment FIREBASE_SECRET is set correctly", "error", err)
		exitCode = 1
		return
	}

	authClient, err := firebaseApp.Auth(context.Background())
	if err != nil {
		slog.Error("Starting application failed, cannot initialize auth client", "error", err)
		exitCode = 1
		return
	}

	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		slog.Error("Starting application failed, DATABASE_URL is not set")
		exitCode = 1
		return
	}

	database, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		slog.Error("Starting application failed, cannot open database", "error", err)
		exitCode = 1
		return
	}

	sqlDB, err := database.DB()
	if err != nil {
		slog.Error("Starting application failed, cannot get database instance", "error", err)
		exitCode = 1
		return
	}

	if err := runDatabaseMigrations(sqlDB); err != nil {
		slog.Error("Starting application failed, database migrations failed", "error", err)
		exitCode = 1
		return
	}

	maxOpenConns := getEnvAsInt("DB_MAX_OPEN_CONNS", 25)
	maxIdleConns := getEnvAsInt("DB_MAX_IDLE_CONNS", 5)
	connMaxLifetime := getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	connMaxIdleTime := getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 10*time.Minute)

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	slog.Info("Database connection pool configured", "maxOpenConns", maxOpenConns, "maxIdleConns", maxIdleConns, "connMaxLifetime", connMaxLifetime, "connMaxIdleTime", connMaxIdleTime)

	dbChecker := healthpkg.NewDatabaseChecker(database, 500*time.Millisecond)
	firebaseChecker := healthpkg.NewFirebaseChecker(authClient, 500*time.Millisecond)
	healthService := healthpkg.NewService(dbChecker, firebaseChecker)

	openAPIConfig, err := os.ReadFile(filepath.Join("openapi", "openapi.yaml"))
	if err != nil {
		slog.Error("Could not read openapi.yaml", "error", err)
		exitCode = 1
		return
	}

	swaggerDocs, err := os.ReadFile(filepath.Join("openapi", "swagger.html"))
	if err != nil {
		slog.Error("Could not read swagger.html", "error", err)
		exitCode = 1
		return
	}

	router := routes.SetupRouter(database, authClient, healthService, openAPIConfig, swaggerDocs)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300, // 5 minutes
	})

	mainServer := &http.Server{
		Addr:         ":8080",
		Handler:      corsHandler.Handler(router),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	metricsRouter := http.NewServeMux()
	metricsRouter.Handle("GET /metrics", promhttp.Handler())

	metricsCors := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"OPTIONS", "GET"},
		MaxAge:         300, // 5 minutes
	})

	metricsServer := &http.Server{
		Addr:         ":9090",
		Handler:      metricsCors.Handler(metricsRouter),
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

	healthService.StartDraining()

	slog.Info("Waiting for load balancer to drain...")
	time.Sleep(5 * time.Second)

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

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
