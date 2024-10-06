package main

import "github.com/henok321/knobel-manager-service/pkg/app"

func main() {
	app := &app.App{}
	app.Initialize()
	app.Router.Run("0.0.0.0:8080")
}
