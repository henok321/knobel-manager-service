package firebaseauth

import (
	"context"
	fbadmin "firebase.google.com/go/v4"
	"google.golang.org/api/option"
	"log"
	"os"
)

var App *fbadmin.App

func InitFirebase() {

	firebaseSecret := []byte(os.Getenv("FIREBASE_SECRET"))
	opt := option.WithCredentialsJSON(firebaseSecret)
	app, err := fbadmin.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("error initializing Firebase app: %v", err)
	}

	App = app
}
