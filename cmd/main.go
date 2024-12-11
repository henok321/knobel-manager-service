package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	firebase "firebase.google.com/go/v4"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"google.golang.org/api/option"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/internal/app"
)

func init() {
	switch os.Getenv("ENVIRONMENT") {
	case "local":
		logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
		slog.SetDefault(slog.New(logHandler))
	default:
		logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
		slog.SetDefault(slog.New(logHandler))
	}
}

func main() {
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	slog.Info("Initialize application")

	firebaseSecret := []byte(os.Getenv("FIREBASE_SECRET"))
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

	databaseUrl := os.Getenv("DATABASE_URL")
	database, err := gorm.Open(postgres.Open(databaseUrl), &gorm.Config{})

	if err != nil {
		slog.Error("Starting application failed, cannot connect to database", "databaseUrl", databaseUrl, "error", err)
		exitCode = 1
		return
	}

	appInstance := app.App{
		Database:   database,
		Router:     http.NewServeMux(),
		AuthClient: authClient,
	}

	appInstance.Initialize()

	appServer := &http.Server{
		Addr:         ":8080",
		Handler:      appInstance.Router,
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("Starting app server", "port", 8080)
		if err := appServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("App server error", "error", err)
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

	if err := appServer.Shutdown(ctx); err != nil {
		slog.Error("Main server shutdown failed", "error", err)
	}
	if err := metricsServer.Shutdown(ctx); err != nil {
		slog.Error("Metrics server shutdown failed", "error", err)
	}

	slog.Info("Servers exited")
}
