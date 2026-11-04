package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const authFailedURL = "/?auth=failed"

// loginHandler sends the user to Pinterest. The state ties the callback to this
// browser, so nobody can trick a user into logging in as someone else.
func loginHandler(c fiber.Ctx) error {
	state := uuid.NewString()
	c.Cookie(sessionCookie(oauthStateCookie, state, time.Now().Add(10*time.Minute)))
	return c.Redirect().To(app.PinClient.GetAuthUri(state))
}

func logoutHandler(c fiber.Ctx) error {
	clearTokens(c)
	clearCookie(c, taskCookie)
	return c.Redirect().To("/")
}

// authRedirectHandler is where Pinterest returns the user, with ?code=&state= on
// success or ?error= if they declined.
func authRedirectHandler(c fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")
	expectedState := c.Cookies(oauthStateCookie)

	clearCookie(c, oauthStateCookie)

	if code == "" || state == "" || state != expectedState {
		log.Printf("OAuth callback rejected (error=%q, has code: %t, state matches: %t)", c.Query("error"), code != "", state != "" && state == expectedState)
		return c.Redirect().To(authFailedURL)
	}

	ctx, cancel := context.WithTimeout(c.Context(), requestTimeout)
	defer cancel()

	tokens, err := app.PinClient.FetchAccessToken(ctx, code)
	if err != nil {
		log.Printf("Token exchange failed: %v", err)
		return c.Redirect().To(authFailedURL)
	}

	writeTokens(c, tokens)

	return c.Redirect().To("/")
}
