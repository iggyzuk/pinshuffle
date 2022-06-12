package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func indexHandler(c *fiber.Ctx) error {

	tm = NewTemplateModel(client.GetAuthUri())

	if mock {

		// Mock: create fake data, process url.

		var mockBoards = make(map[string]*Board)

		mockBoards["visual-style"] = &Board{Id: "visual-style", Name: "Visual Style"}
		mockBoards["ideas"] = &Board{Id: "ideas", Name: "Ideas"}
		mockBoards["concepts"] = &Board{Id: "concepts", Name: "Concepts"}

		tm.Mock(c.Context().URI(), mockBoards)

		// No need to use the randomizer – we forcefull put template-pins into the template-model.

	} else {

		var cookie = c.Cookies("access_token")

		if len(cookie) == 0 {
			log.Println("Missing Cookie")
			tm.Message = "💭 Waiting for access to Pinterest account"
			tm.Authenticated = false
		} else {
			log.Println("Cookie Exists")
			client.AccessToken = cookie
			tm.Authenticated = true

			user, userErr := client.FetchUserAccount()
			if userErr != nil {
				tm.Error = userErr.Error()
			}

			tm.User = &TemplateUser{Name: user.Username, IconURL: user.ProfileImage, URL: user.WebsiteURL}

			// Real: fetch boards, process url, randomize.

			var clientBoards = make(map[string]*Board)

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

			parseErr := tm.ParseUrlQueries(c.Context().URI(), clientBoards)
			if parseErr != nil {
				return parseErr
			}

			// Randomize – if there are any url-specified boards.
			if len(tm.UrlQuery.Boards) > 0 {

				randomizedPins := NewRandomizer(clientBoards).GetRandomizedPins(tm.UrlQuery.Max, tm.UrlQuery.Boards)

				for _, randomizedPin := range randomizedPins {
					tm.Pins = append(tm.Pins, TemplatePin{
						Id:       randomizedPin.Id,
						Name:     randomizedPin.Title,
						Color:    randomizedPin.DominantColor,
						ImageURL: GetImageResolution(tm.UrlQuery.ImageResolution, randomizedPin.Media.Images).Url,
					})
				}
			}
		}
	}

	// Render the HTML page.
	return c.Render("layout", tm)
}
