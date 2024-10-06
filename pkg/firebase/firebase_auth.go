package firebaseauth

import (
	"context"
	fbadmin "firebase.google.com/go/v4"
	"log"
)

var App *fbadmin.App

func InitFirebase() {
	app, err := fbadmin.NewApp(context.Background(), nil)
	if err != nil {
		log.Fatalf("error initializing Firebase app: %v", err)
	}

	App = app
}
