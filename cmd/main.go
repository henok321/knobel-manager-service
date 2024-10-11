package main

import (
	"github.com/henok321/knobel-manager-service/internal/app"
)

func main() {
	instance := &app.App{}
	instance.Initialize()
	err := instance.Router.Run("0.0.0.0:8080")
	if err != nil {
		return
	}
}
