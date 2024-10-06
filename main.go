package main

func main() {
	app := &App{}
	app.Initialize()
	app.Router.Run("0.0.0.0:8080")
}
