package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	fbadmin "firebase.google.com/go/v4"
	"google.golang.org/api/option"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/internal/app"
)

func init() {
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(logHandler))
}

func main() {
	slog.Info("Initialize application")

	firebaseAdmin := []byte(os.Getenv("FIREBASE_SECRET"))
	firebaseConfig := option.WithCredentialsJSON(firebaseAdmin)
	firebaseApp, err := fbadmin.NewApp(context.Background(), nil, firebaseConfig)

	if err != nil {
		slog.Error("Starting application failed, cannot initialize firebase client", "error", err)
		os.Exit(1)
	}

	authClient, err := firebaseApp.Auth(context.Background())

	if err != nil {
		slog.Error("Starting application failed, cannot initialize auth client", "error", err)
		os.Exit(1)
	}

	databaseUrl := os.Getenv("DATABASE_URL")
	database, err := gorm.Open(postgres.Open(databaseUrl), &gorm.Config{})

	if err != nil {
		slog.Error("Starting application failed, cannot connect to database", "databaseUrl", databaseUrl, "error", err)
		os.Exit(1)
	}

	appInstance := app.App{
		Database:       database,
		Router:         http.NewServeMux(),
		AuthMiddleware: middleware.NewAuthenticationMiddleware(authClient),
	}

	appInstance.Initialize()

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", 8080),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
		Handler:      appInstance.Router,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("Starting server", "port", 8080)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Error starting server", "error", err)
			os.Exit(1)
		}
	}()

	<-sigChan
	slog.Info("Shutdown signal received, shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
	}

	slog.Info("Server exiting")
}
