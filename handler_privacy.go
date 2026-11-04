package main

import "github.com/gofiber/fiber/v3"

func privacyHandler(c fiber.Ctx) error {
	return c.Render("privacy", nil)
}
