package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

type StrategyFunc func(c *fiber.Ctx) (TemplateModel, error)

func GetMockTemplateModel(c *fiber.Ctx) (TemplateModel, error) {

	var tmplController = NewTemplateController("")

	tmplController.Mock(c.Context().URI())

	return tmplController.Model, nil
}

func GetTemplateModel(c *fiber.Ctx) (TemplateModel, error) {

	var tmplController = NewTemplateController(app.PinClient.GetAuthUri())

	var accessToken = c.Cookies("access_token")

	if len(accessToken) == 0 {

		// Real: but not authenticated, just show an error.

		log.Println("Missing Cookie")
		tmplController.Model.Message = "💭 Waiting for access to Pinterest account"
		tmplController.Model.Authenticated = false
	} else {

		// Real: fetch boards, process url, randomize.

		log.Println("Cookie Exists")
		app.PinClient.AccessToken = accessToken
		tmplController.Model.Authenticated = true

		user, userErr := app.PinClient.FetchUserAccount()
		if userErr != nil {
			tmplController.Model.Error = userErr.Error()
		} else {
			tmplController.Model.User = TemplateUser{
				Name:    user.Username,
				IconURL: user.ProfileImage,
				URL:     user.WebsiteURL,
			}
		}

		var clientBoards = make(map[string]Board)

		var fetchedClientBoards, err = app.PinClient.FetchBoards()

		if err != nil {
			tmplController.Model.Error = err.Error()
		} else {
			for _, board := range fetchedClientBoards {
				// make a deep copy from a pointer
				clientBoards[board.Id] = Board{Id: board.Id, Name: board.Name}
			}
		}

		if len(clientBoards) == 0 {
			tmplController.Model.Message = "💭 No boards found"
		}

		parseErr := tmplController.ParseUrlQueries(c.Context().URI(), clientBoards)
		if parseErr != nil {
			tmplController.Model.Error = parseErr.Error()
		}

		// Find task id in cookies.
		taskId := c.Cookies("task")

		if len(taskId) > 0 {

			task, exists := app.Tasks[taskId]

			// Copy all pins when the task completes
			if exists && task.IsComplete {

				for _, randomizedPin := range task.Pins {
					tmplController.AddPin(&randomizedPin)
				}

				// delete it and clear the cookie
				delete(app.Tasks, taskId)
				c.ClearCookie("task")
			} else {

				// Let the client know that task is there and is still processing...
				tmplController.Model.Message = fmt.Sprintf("Still Processing Task with Id: %s", taskId)
				tmplController.Model.Loading = true
			}

			// Check and delete any abandoned tasks
			abandonedTaskIds := []string{}
			now := time.Now()

			for _, taskToCheck := range app.Tasks {
				elapsed := now.Sub(taskToCheck.Timestamp)
				fmt.Printf("Checking Task; Elapsed: %s\n", elapsed)
				if elapsed >= 5*time.Minute {
					abandonedTaskIds = append(abandonedTaskIds, taskToCheck.Id)
				}
			}

			for _, abandonedTaskId := range abandonedTaskIds {
				delete(app.Tasks, abandonedTaskId)
				fmt.Printf("Deleted Abandoned Task: %s\n", abandonedTaskId)
			}

			return tmplController.Model, nil

		} else if len(tmplController.Model.UrlQuery.Boards) > 0 {

			// Create a new task and let the client know of its id
			task := NewTask()

			cookie := fiber.Cookie{
				Name:    "task",
				Value:   task.Id,
				Expires: time.Now().Add(5 * time.Minute),
			}

			c.Cookie(&cookie)

			// Start the task in the background and send the response back to the client.
			go task.Process(app.PinClient, clientBoards, tmplController.Model.UrlQuery)
			tmplController.Model.Loading = true
			return tmplController.Model, nil
		}
	}

	// No task currently processing, no boards selected
	return tmplController.Model, nil
}
