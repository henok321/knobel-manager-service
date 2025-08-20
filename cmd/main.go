package main

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/henok321/knobel-manager-service/api/logging"

	"github.com/rs/cors"

	firebase "firebase.google.com/go/v4"

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

func main() {
	exitCode := 0

	defer func() {
		os.Exit(exitCode)
	}()

	slog.Info("Initialize application")

	firebaseSecret, err := base64.RawStdEncoding.DecodeString(os.Getenv("FIREBASE_SECRET"))
	if err != nil {
		slog.Error("Starting application failed, cannot decode FIREBASE_SECRET")
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
	database, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		slog.Error("Starting application failed, cannot connect to database", "databaseUrl", databaseURL, "error", err)
		exitCode = 1
		return
	}

	router := routes.SetupRouter(database, authClient)

	mainServer := &http.Server{
		Addr:         ":8080",
		Handler:      cors.AllowAll().Handler(router),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	metricsRouter := http.NewServeMux()
	metricsRouter.Handle("GET /metrics", promhttp.Handler())

	metricsServer := &http.Server{
		Addr:         ":9090",
		Handler:      cors.AllowAll().Handler(metricsRouter),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

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

	<-sigChan
	slog.Info("Shutdown signal received, shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := mainServer.Shutdown(ctx); err != nil {
		slog.Error("Main server shutdown failed", "error", err)
	}

	if err := metricsServer.Shutdown(ctx); err != nil {
		slog.Error("Metrics server shutdown failed", "error", err)
	}

	slog.Info("Servers exited")
}
