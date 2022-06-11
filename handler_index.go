package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func indexHandler(c *fiber.Ctx) error {

	tm = NewTemplateModel(client.GetAuthUri())

	var cookie = c.Cookies("access_token")

	if len(cookie) == 0 {
		log.Println("Missing Cookie")
		tm.Message = "Waiting for access to Pinterest account."
		tm.Authenticated = false
	} else {
		log.Println("Cookie Exists")
		client.AccessToken = cookie
		tm.Authenticated = true

		var templateBoards []TemplateBoard
		for _, b := range client.FetchBoards().Items {
			templateBoards = append(templateBoards, TemplateBoard{
				Name:     b.Name,
				PinCount: 16, // TODO: is this still possible?
			})
		}
		tm.Boards = templateBoards
	}

	err := tm.ParseUrlQueries(c.Context().URI())
	if err != nil {
		return err
	}

	// tm.Mock()

	randomizer = NewRandomizer(tm.UrlQuery)

	// Render the HTML page.
	return c.Render("layout", tm)
}
