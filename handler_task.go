package main

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

func taskHandler(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")

	task, exists := app.Tasks.Get(strings.TrimSpace(c.Params("*")))
	if !exists {
		return c.JSON(TaskProgress{Status: "error"})
	}

	return c.JSON(task.Progress())
}
