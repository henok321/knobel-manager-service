package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/internal/app"

	firebaseauth "github.com/henok321/knobel-manager-service/internal/auth"
)

func init() {
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(logHandler))
}

func main() {
	slog.Info("Starting application ...")
	firebaseauth.InitFirebase()

	url := os.Getenv("DATABASE_URL")
	database, err := gorm.Open(postgres.Open(url), &gorm.Config{})

	if err != nil {
		slog.Error("Starting application failed, cannot start connect to database", "error", err)
	}

	appInstance := app.App{
		Database:       database,
		Router:         http.NewServeMux(),
		AuthMiddleware: middleware.Authentication,
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
	}
}
