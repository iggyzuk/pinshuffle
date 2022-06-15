package main

import (
	"github.com/gofiber/fiber/v2"
)

func indexHandler(c *fiber.Ctx) error {

	model, err := strategyFunc(c.Context().URI(), c.Cookies("access_token"))

	if err != nil {
		return err
	}

	// Render the HTML page.
	return c.Render("layout", model)
}
