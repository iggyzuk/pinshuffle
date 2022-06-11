package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

var clientBoards map[string]*Board

func indexHandler(c *fiber.Ctx) error {

	clientBoards = make(map[string]*Board)

	tm = NewTemplateModel(client.GetAuthUri())

	var cookie = c.Cookies("access_token")

	if len(cookie) == 0 {
		log.Println("Missing Cookie")
		tm.Message = "💭 Waiting for access to Pinterest account"
		tm.Authenticated = false
	} else {
		log.Println("Cookie Exists")
		client.AccessToken = cookie
		tm.Authenticated = true

		var fetchedClientBoards, err = client.FetchBoards()
		if err != nil {
			tm.Error = err.Error()
		} else {
			for _, board := range fetchedClientBoards.Items {
				clientBoards[board.Id] = &Board{Id: board.Id, Name: board.Name} // TODO: why do we need to copy it? (probably cause it's an iterator value)
			}
		}

		if len(clientBoards) == 0 {
			tm.Message = "💭 No boards found"
		}

		// tm.Mock()

		parseErr := tm.ParseUrlQueries(c.Context().URI())
		if parseErr != nil {
			return parseErr
		}

		if len(tm.UrlQuery.Boards) > 0 {

			randomizedPins := NewRandomizer().GetRandomizedPins(tm.UrlQuery.Max, tm.UrlQuery.Boards)

			for _, randomizedPin := range randomizedPins {
				tm.Pins = append(tm.Pins, TemplatePin{
					ImageURL: GetImageResolution(tm.UrlQuery.ImageResolution, randomizedPin.Media.Images).Url,
					PinURL:   "#",
					Color:    randomizedPin.Color,
				})
			}
		}
	}

	// Render the HTML page.
	return c.Render("layout", tm)
}
