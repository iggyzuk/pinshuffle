package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
)

// Fonts are imported by the themes, scripts and privacy page styles come from CDNs,
// pin images from Pinterest. Inline styles are for the dominant color of each pin.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self' https://cdn.jsdelivr.net https://cdnjs.cloudflare.com; " +
	"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://fonts.googleapis.com; " +
	"font-src 'self' data: https://fonts.gstatic.com; " +
	"img-src 'self' data: https:; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

type App struct {
	PinClient *PinterestClient
	Func      StrategyFunc
	Tasks     *TaskStore
}

func NewApp() App {
	app := App{
		PinClient: NewClient(),
		Func:      GetTemplateModel,
		Tasks:     NewTaskStore(),
	}
	// Use real or mock index strategy when you find "-mock" argument
	if len(os.Args) > 1 && os.Args[1] == "-mock" {
		app.Func = GetMockTemplateModel
	}
	return app
}

var app App

func securityHeaders(c *fiber.Ctx) error {
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")

	// The monitor page is the only one that needs inline scripts.
	if !strings.HasPrefix(c.Path(), "/monitor") {
		c.Set(fiber.HeaderContentSecurityPolicy, contentSecurityPolicy)
	}

	return c.Next()
}

func main() {

	// Must come before NewApp, which reads the environment.
	godotenv.Load(".env")

	app = NewApp()

	// Initialize standard Go html template engine
	engine := html.New("./templates", ".gohtml")
	engine.AddFunc("IsBoardSelected", IsBoardSelected)
	engine.AddFunc("SortBoards", SortBoards)

	server := fiber.New(fiber.Config{
		Views: engine,
		// Fiber reuses the memory behind strings like cookies between requests,
		// which would corrupt anything a background task holds on to.
		Immutable:    true,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  2 * time.Minute,
	})

	server.Use(recover.New())
	server.Use(securityHeaders)
	server.Use(logger.New())
	server.Use(compress.New())

	// Server stats aren't for the public: they need MONITOR_PASSWORD to be set.
	if password := os.Getenv("MONITOR_PASSWORD"); password != "" {
		server.Use("/monitor", basicauth.New(basicauth.Config{
			Users: map[string]string{"admin": password},
		}), monitor.New())
	}

	// Load static files like CSS, Images & JavaScript.
	server.Static("/static", "./static")

	server.Get("/", indexHandler)
	server.Get("/login", loginHandler)
	server.Post("/logout", logoutHandler)
	server.Get("/redirect", authRedirectHandler)
	server.Get("/task/*", taskHandler)
	server.Get("/privacy", privacyHandler)

	// 404 handler.
	server.Use(func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNotFound)
	})

	// Throw away results that nobody came back for.
	go func() {
		for now := range time.Tick(time.Minute) {
			app.Tasks.DeleteExpired(now)
		}
	}()

	// Finish requests in flight when Fly stops the machine.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		if err := server.ShutdownWithTimeout(10 * time.Second); err != nil {
			log.Printf("Shutdown: %v", err)
		}
	}()

	// Get port from env vars.
	var port = os.Getenv("PORT")

	// Use a default port if none was set in env.
	if port == "" {
		port = "8275"
	}

	// Start server on http://${fly-url}:${port}
	if err := server.Listen(":" + port); err != nil {
		log.Fatal(err)
	}
}
