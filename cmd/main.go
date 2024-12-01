package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
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
	exitCode := 0

	defer func() {
		os.Exit(exitCode)
	}()

	slog.Info("Initialize application")

	firebaseAdmin := []byte(os.Getenv("FIREBASE_SECRET"))
	opt := option.WithCredentialsJSON(firebaseAdmin)
	firebaseApp, err := fbadmin.NewApp(context.Background(), nil, opt)

	if err != nil {
		slog.Error("Starting application failed, cannot initialize firebase client", "error", err)
		exitCode = 1
		return
	}

	authClient, err := firebaseApp.Auth(context.Background())

	if err != nil {
		slog.Error("Starting application failed, cannot initialize auth client", "error", err)
		exitCode = 1
		return
	}

	url := os.Getenv("DATABASE_URL")
	database, err := gorm.Open(postgres.Open(url), &gorm.Config{})

	if err != nil {
		slog.Error("Starting application failed, cannot connect to database", "error", err)
		exitCode = 1
		return
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

	slog.Info("Starting server", "port", 8080)

	if err := server.ListenAndServe(); err != nil {
		slog.Error("Error starting server", "error", err)
		exitCode = 1
		return
	}
}
