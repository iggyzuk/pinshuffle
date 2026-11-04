package main

import (
	"errors"
	"log"
	"net/url"

	"github.com/gofiber/fiber/v3"
)

// Showing sections takes a request for every board, and the page reloads whenever
// new pins are picked. This way that doesn't ask Pinterest all over again.
const sectionsCacheControl = "private, max-age=300"

// sectionsHandler lists a board's sections for the boards modal.
func sectionsHandler(c fiber.Ctx) error {
	boardId, err := url.PathUnescape(c.Params("id"))
	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	sections, err := app.Sections(c, boardId)

	switch {
	case errors.Is(err, ErrUnauthorized):
		return c.SendStatus(fiber.StatusUnauthorized)
	case err != nil:
		log.Printf("Loading sections of board %s failed: %v", boardId, err)
		return c.SendStatus(fiber.StatusBadGateway)
	}

	c.Set(fiber.HeaderCacheControl, sectionsCacheControl)

	if sections == nil {
		sections = []Section{}
	}
	return c.JSON(sections)
}
