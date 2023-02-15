package main

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/template/html"
	"github.com/joho/godotenv"
)

var strategyFunc StrategyFunc

// String (Id) -> *Task
var tasks map[string]*Task

func main() {

	tasks = make(map[string]*Task)

	// Use real or mock index strategy when you find "-mock" argument
	strategyFunc = GetTemplateModel

	if len(os.Args) > 1 {
		if os.Args[1] == "-mock" {
			strategyFunc = GetMockTemplateModel
		}
	}

	godotenv.Load(".env")

	// Initialize standard Go html template engine
	engine := html.New("./templates", ".gohtml")
	engine.AddFunc("IsBoardSelected", IsBoardSelected)
	engine.AddFunc("SortBoards", SortBoards)
	engine.AddFunc("Iterate", Iterate)

	// Delims sets the action delimiters to the specified strings
	engine.Delims("{{", "}}") // Optional. Default: engine delimiters

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Use(func(c *fiber.Ctx) error {
		c.Set("Content-Security-Policy", "default-src *; frame-src *; script-src 'self' 'unsafe-inline';")
		return c.Next()
	})

	app.Use(logger.New())
	app.Use("/monitor", monitor.New())

	// Load static files like CSS, Images & JavaScript.
	app.Static("/static", "./static")

	app.Get("/", indexHandler)
	app.Get("/redirect", authRedirectHandler)
	app.Get("/task/*", taskHandler)
	app.Get("/privacy", privacyHandler)

	// 404 handler.
	app.Use(func(c *fiber.Ctx) error {
		return c.SendStatus(404) // => 404 "Not Found"
	})

	// Get port from env vars.
	var port = os.Getenv("PORT")

	// Use a default port if none was set in env.
	if port == "" {
		port = "8080"
	}

	// Start server on http://${fly-url}:${port}
	app.Listen(":" + port)
}
