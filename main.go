package main

import (
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html"
	"github.com/joho/godotenv"
)

// TemplateData is the main object we pass for templating HTML
type TemplateData struct {
	OAuthURL      string
	Authenticated bool
	Boards        []TemplateBoard
	Error         string
	Message       string
}

type TemplateBoard struct {
	Name     string
	PinCount int
}

var client *PinterestClient

var tlsCertPath = os.Getenv("TLS_CERT_PATH")
var tlsKeyPath = os.Getenv("TLS_KEY_PATH")

func main() {

	godotenv.Load(".env")

	client = NewClient(os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET"))

	// Initialize standard Go html template engine
	engine := html.New("./templates", ".gohtml")

	// Delims sets the action delimiters to the specified strings
	engine.Delims("{{", "}}") // Optional. Default: engine delimiters

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Use(logger.New())

	// Load static files like CSS, Images & JavaScript.
	app.Static("/static", "./static")

	app.Get("/", indexHandler)
	app.Get("/redirect", authRedirectHandler)

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

func indexHandler(c *fiber.Ctx) error {

	templateData := TemplateData{
		OAuthURL:      client.GetAuthUri(),
		Authenticated: false,
		Boards:        nil,
		Error:         "",
		Message:       "",
	}

	var cookie = c.Cookies("access_token")

	if len(cookie) == 0 {
		log.Println("Missing Cookie")
	} else {
		log.Println("Cookie Exists")
		client.AccessToken = cookie
		templateData.Authenticated = true

		var templateBoards []TemplateBoard
		for _, b := range client.FetchBoards().Items {
			templateBoards = append(templateBoards, TemplateBoard{
				Name:     b.Name,
				PinCount: 16, // TODO: is this still possible?
			})
		}
		templateData.Boards = templateBoards
	}

	// Render the HTML page.
	return c.Render("layout", templateData)
}

func authRedirectHandler(c *fiber.Ctx) error {

	// Once the user approves authorization for your app, they will be sent to your redirect URI as indicated in the request.
	// 		We will add the following parameters as we make the call to your redirect URI:
	//			code: This is the code you will use in the next step to exchange for an access token.
	//			state: This is the optional parameter to prevent cross-site request forgery. Check to make sure it matches what was passed in the first step of the flow. If it does not, the exchange may have been subject to a cross-site request forgery attack.
	// A redirect URI such as https://www.example.com/oauth/pinterest/oauth_response/
	// 		will result in a callback request like: https://www.example.com/oauth/pinterest/oauth_response/?code={CODE}&state={YOUR_OPTIONAL_STRING}

	codeKey := c.Query("code")

	if len(codeKey) > 0 {
		log.Println("Code Key: " + codeKey)

		err := client.FetchAuthToken(codeKey)

		if err != nil {
			return err
		}

		cookie := fiber.Cookie{
			Name:    "access_token",
			Value:   client.AccessToken,
			Expires: time.Now().Add(365 * 24 * time.Hour),
		}

		c.Cookie(&cookie)

		log.Printf("Cookie: %s, value: %s", cookie.Name, cookie.Value)

		log.Println("Success! Go back home!")

		c.Redirect(client.MainURL)
	}

	return nil
}
