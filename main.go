package main

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/template/html"
	"github.com/joho/godotenv"
)

type App struct {
	PinClient *PinterestClient
	Func      StrategyFunc
	Tasks     map[string]*Task
}

func NewApp() App {
	app := App{
		PinClient: NewClient(),
		Func:      GetTemplateModel,
		Tasks:     make(map[string]*Task),
	}
	// Use real or mock index strategy when you find "-mock" argument
	if len(os.Args) > 1 {
		if os.Args[1] == "-mock" {
			app.PinClient = nil
			app.Func = GetMockTemplateModel
		}
	}
	return app
}

var app App

func main() {

	app = NewApp()

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

	// todo: use a better content-secutiy-policy.
	app.Use(func(c *fiber.Ctx) error {
		c.Set("Content-Security-Policy", "default-src * 'unsafe-inline' 'unsafe-eval'; script-src * 'unsafe-inline' 'unsafe-eval'; connect-src * 'unsafe-inline'; img-src * data: blob: 'unsafe-inline'; frame-src *; child-src *; style-src * 'unsafe-inline'; font-src * 'unsafe-inline'; manifest-src *; navigate-to *;")
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
		port = "8275"
	}

	// Start server on http://${fly-url}:${port}
	app.Listen(":" + port)
}
