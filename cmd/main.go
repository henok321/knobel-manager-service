package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/internal/app"
	log "github.com/sirupsen/logrus"

	firebaseauth "github.com/henok321/knobel-manager-service/internal/auth"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
}

func main() {
	log.Infoln("Starting application ...")
	firebaseauth.InitFirebase()

	url := os.Getenv("DATABASE_URL")
	database, err := gorm.Open(postgres.Open(url), &gorm.Config{})

	if err != nil {
		log.Fatalln("Starting application failed, cannot start connect to database", err)
	}

	appInstance := app.App{
		DB: database,
	}

	router := appInstance.Initialize(middleware.Authentication)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", 8080),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
		Handler:      router,
	}

	log.Infof("Starting server on port %d", 8080)

	if err := server.ListenAndServe(); err != nil {
		log.WithError(err).Fatal("Error starting server")
	}
}
