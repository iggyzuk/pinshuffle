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
		for _, board := range client.FetchBoards().Items {
			templateBoards = append(templateBoards, TemplateBoard{
				Board: &board,
				Name:  board.Name,
				Id:    board.Id,
			})
		}
		tm.Boards = templateBoards

		// tm.Mock()

		err := tm.ParseUrlQueries(c.Context().URI())
		if err != nil {
			return err
		}

		randomizedPins := NewRandomizer().GetRandomizedPins(tm)

		for _, rp := range randomizedPins {
			tm.Pins = append(tm.Pins, TemplatePin{
				ImageURL: rp.Media.Images.Medium.Url,
				PinURL:   "#",
				Color:    rp.Color,
			})
		}
	}

	// Render the HTML page.
	return c.Render("layout", tm)
}
