package main

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/template/html"
	"github.com/joho/godotenv"
)

var mock = false

func main() {

	if len(os.Args) > 1 {
		if os.Args[1] == "-mock" {
			mock = true
		}
	}

	godotenv.Load(".env")

	// Initialize standard Go html template engine
	engine := html.New("./templates", ".gohtml")
	engine.AddFunc("isBoardSelected", IsBoardSelected)
	engine.AddFunc("Iterate", Iterate)

	// Delims sets the action delimiters to the specified strings
	engine.Delims("{{", "}}") // Optional. Default: engine delimiters

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Use(logger.New())
	app.Use("/monitor", monitor.New())

	// Load static files like CSS, Images & JavaScript.
	app.Static("/static", "./static")

	app.Get("/", indexHandler)
	app.Get("/redirect", authRedirectHandler)
	app.Get("/privacy", privacyHandler)

	// 404 handler.
	app.Use(func(c *fiber.Ctx) error {
		return c.SendStatus(404) // => 404 "Not Found"
	})

	// Get port from env vars.
	var port = os.Getenv("PORT")

	// Use a default port if none was set in env.
	if port == "" {
		port = "3000"
	}

	// Start server on http://${heroku-url}:${port}
	app.Listen(":" + port)
}
