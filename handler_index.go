package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func indexHandler(c *fiber.Ctx) error {

	var pinClient = NewClient()
	var tmplController = NewTemplateController(pinClient.GetAuthUri())

	// TODO: every time anyone goes to / we check this? can we use a strategy pattern?

	if mock {

		// Mock: create fake data, process url.

		tmplController.Mock(c.Context().URI())

	} else {

		var cookie = c.Cookies("access_token")

		if len(cookie) == 0 {

			// Real: but not authenticated, just show an error.

			log.Println("Missing Cookie")
			tmplController.Model.Message = "💭 Waiting for access to Pinterest account"
			tmplController.Model.Authenticated = false
		} else {

			// Real: fetch boards, process url, randomize.

			log.Println("Cookie Exists")
			pinClient.AccessToken = cookie
			tmplController.Model.Authenticated = true

			user, userErr := pinClient.FetchUserAccount()
			if userErr != nil {
				tmplController.Model.Error = userErr.Error()
			} else {
				tmplController.Model.User = TemplateUser{
					Name:    user.Username,
					IconURL: user.ProfileImage,
					URL:     user.WebsiteURL,
				}
			}

			var clientBoards = make(map[string]*Board)

			var fetchedClientBoards, err = pinClient.FetchBoards()

			if err != nil {
				tmplController.Model.Error = err.Error()
			} else {
				for _, board := range fetchedClientBoards.Items {
					clientBoards[board.Id] = &Board{Id: board.Id, Name: board.Name} // TODO: why do we need to copy it? (probably cause it's an iterator value)
				}
			}

			if len(clientBoards) == 0 {
				tmplController.Model.Message = "💭 No boards found"
			}

			parseErr := tmplController.ParseUrlQueries(c.Context().URI(), clientBoards)
			if parseErr != nil {
				return parseErr
			}

			// Randomize – if there are any url-specified boards.
			if len(tmplController.Model.UrlQuery.Boards) > 0 {

				randomizer := NewRandomizer(pinClient, clientBoards)
				randomizedPins := randomizer.GetRandomizedPins(tmplController.Model.UrlQuery.Max, tmplController.Model.UrlQuery.Boards)

				for _, randomizedPin := range randomizedPins {
					tmplController.AddPin(&randomizedPin)
				}
			}
		}
	}

	// Render the HTML page.
	return c.Render("layout", tmplController.Model)
}
