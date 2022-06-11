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
		tm.Message = "Waiting for access to Pinterest account."
		tm.Authenticated = false
	} else {
		log.Println("Cookie Exists")
		client.AccessToken = cookie
		tm.Authenticated = true

		for _, board := range client.FetchBoards().Items {
			clientBoards[board.Id] = &Board{Id: board.Id, Name: board.Name} // TODO: why do we need to copy it?
		}

		// tm.Mock()

		err := tm.ParseUrlQueries(c.Context().URI())
		if err != nil {
			return err
		}

		if len(tm.UrlQuery.Boards) > 0 {

			randomizedPins := NewRandomizer().GetRandomizedPins(tm.UrlQuery.Max, tm.UrlQuery.Boards)

			for _, randomizedPin := range randomizedPins {
				tm.Pins = append(tm.Pins, TemplatePin{
					ImageURL: GetImageSize(tm.UrlQuery.ImageSize, randomizedPin.Media.Images).Url,
					PinURL:   "#",
					Color:    randomizedPin.Color,
				})
			}
		}
	}

	// Render the HTML page.
	return c.Render("layout", tm)
}

func GetImageSize(imageSize int, images Images) Image {
	if imageSize == 0 {
		return images.Small
	}
	if imageSize == 1 {
		return images.Medium
	}
	if imageSize == 2 {
		return images.Huge
	}
	if imageSize == 3 {
		return images.Original
	}
	return images.Medium
}
