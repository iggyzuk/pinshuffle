package main

import (
	"log"

	"github.com/valyala/fasthttp"
)

type StrategyFunc func(uri *fasthttp.URI, accessToken string) (TemplateModel, error)

func GetMockTemplateModel(uri *fasthttp.URI, accessToken string) (TemplateModel, error) {

	var tmplController = NewTemplateController("")

	tmplController.Mock(uri)

	return tmplController.Model, nil
}

func GetTemplateModel(uri *fasthttp.URI, accessToken string) (TemplateModel, error) {

	var pinClient = NewClient()

	var tmplController = NewTemplateController(pinClient.GetAuthUri())

	if len(accessToken) == 0 {

		// Real: but not authenticated, just show an error.

		log.Println("Missing Cookie")
		tmplController.Model.Message = "💭 Waiting for access to Pinterest account"
		tmplController.Model.Authenticated = false
	} else {

		// Real: fetch boards, process url, randomize.

		log.Println("Cookie Exists")
		pinClient.AccessToken = accessToken
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
			for _, board := range fetchedClientBoards {
				clientBoards[board.Id] = &Board{Id: board.Id, Name: board.Name} // TODO: why do we need to copy it? (probably cause it's an iterator value)
			}
		}

		if len(clientBoards) == 0 {
			tmplController.Model.Message = "💭 No boards found"
		}

		parseErr := tmplController.ParseUrlQueries(uri, clientBoards)
		if parseErr != nil {
			tmplController.Model.Error = parseErr.Error()
		}

		// Randomize – if there are any url-specified boards.
		if len(tmplController.Model.UrlQuery.Boards) > 0 {

			randomizer := NewRandomizer(pinClient, clientBoards)
			randomizedPins, randErr := randomizer.GetRandomizedPins(tmplController.Model.UrlQuery.Max, tmplController.Model.UrlQuery.Boards)

			if randErr != nil {
				tmplController.Model.Error = randErr.Error()
			}

			for _, randomizedPin := range randomizedPins {
				tmplController.AddPin(&randomizedPin)
			}
		}
	}

	return tmplController.Model, nil
}
