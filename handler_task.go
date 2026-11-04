package main

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

func taskHandler(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")

	task, exists := app.Tasks.Get(strings.TrimSpace(c.Params("*")))

	switch {
	case !exists:
		return c.SendString("error")
	case task.IsComplete():
		return c.SendString("complete")
	default:
		return c.SendString("processing")
	}
}
