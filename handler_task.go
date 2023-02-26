package main

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func taskHandler(c *fiber.Ctx) error {
	taskId := c.Params("*")
	taskId = strings.TrimSpace(taskId)

	task, exists := app.Tasks[taskId]

	if exists {
		if task.IsComplete {
			c.SendString("complete")
			return nil
		} else {
			c.SendString("processing")
			return nil
		}
	}
	c.SendString("error")
	return nil
}
