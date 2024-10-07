package main

import (
	"github.com/henok321/knobel-manager-service/internal/app"
)

func main() {
	app := &app.App{}
	app.Initialize()
	err := app.Router.Run("0.0.0.0:8080")
	if err != nil {
		return
	}
}
