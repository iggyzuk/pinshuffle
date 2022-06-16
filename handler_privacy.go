package main

import "github.com/gofiber/fiber/v2"

func privacyHandler(c *fiber.Ctx) error {
	return c.SendString("Pinshuffle uses a cookie for the access token and the selected theme.")
}
