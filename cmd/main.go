package main

import (
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/internal/app"
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

	database, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})

	if err != nil {
		log.Fatalln("Starting application failed, cannot start connect to database", err)
	}

	router := gin.Default()

	instance := &app.App{
		DB:             database,
		Router:         router,
		AuthMiddleware: middleware.Authentication(),
	}
	instance.Initialize()

	err = instance.Router.Run("0.0.0.0:8080")

	if err != nil {
		log.Fatalln("Starting application failed, cannot start router instance", err)
	}
}
