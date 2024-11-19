package main

import (
	"log"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/internal/app"
	firebaseauth "github.com/henok321/knobel-manager-service/internal/auth"
)

func main() {
	firebaseauth.InitFirebase()
	instance := &app.App{}
	instance.Initialize(middleware.AuthMiddleware())
	err := instance.Router.Run("0.0.0.0:8080")
	if err != nil {
		log.Fatalln("Starting application failed, cannot start router instance", err)
	}
}
