package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

const (
	accessTokenCookie  = "access_token"
	refreshTokenCookie = "refresh_token"
	tokenExpiryCookie  = "token_expires"
	oauthStateCookie   = "oauth_state"
	taskCookie         = "task"

	// Refresh this long before the access token (30 days) expires, so a user who
	// returns weekly never hits an expired token.
	refreshMargin = 7 * 24 * time.Hour
)

// Tokens is a user's Pinterest credentials. Pinterest tokens can't be inspected
// for their expiry, so ExpiresAt comes from the token response and is remembered
// in a cookie. It is zero for sessions from before refreshing existed.
type Tokens struct {
	AccessToken      string
	RefreshToken     string
	ExpiresAt        time.Time
	RefreshExpiresAt time.Time
}

func (t Tokens) Empty() bool {
	return t.AccessToken == "" && t.RefreshToken == ""
}

func (t Tokens) Expired(now time.Time) bool {
	return t.AccessToken == "" || (!t.ExpiresAt.IsZero() && !now.Before(t.ExpiresAt))
}

// NeedsRefresh reports whether the access token should be renewed now.
func (t Tokens) NeedsRefresh(now time.Time) bool {
	if t.RefreshToken == "" {
		return false
	}
	return t.AccessToken == "" || (!t.ExpiresAt.IsZero() && now.Add(refreshMargin).After(t.ExpiresAt))
}

func readTokens(c fiber.Ctx) Tokens {
	tokens := Tokens{
		AccessToken:  c.Cookies(accessTokenCookie),
		RefreshToken: c.Cookies(refreshTokenCookie),
	}
	if unix, err := strconv.ParseInt(c.Cookies(tokenExpiryCookie), 10, 64); err == nil {
		tokens.ExpiresAt = time.Unix(unix, 0)
	}
	return tokens
}

// sessionCookie builds a cookie scripts can't read and other sites can't send.
func sessionCookie(name, value string, expires time.Time) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HTTPOnly: true,
		Secure:   strings.HasPrefix(app.PinClient.MainURL, "https://"),
		SameSite: fiber.CookieSameSiteLaxMode,
	}
}

func clearCookie(c fiber.Ctx, name string) {
	c.Cookie(sessionCookie(name, "", time.Unix(1, 0)))
}

func writeTokens(c fiber.Ctx, tokens Tokens) {
	// Keep the access token's cookie alive for a year if its lifetime is unknown,
	// matching how long these cookies have always lasted.
	expires := tokens.ExpiresAt
	if expires.IsZero() {
		expires = time.Now().Add(365 * 24 * time.Hour)
	}
	c.Cookie(sessionCookie(accessTokenCookie, tokens.AccessToken, expires))

	if !tokens.ExpiresAt.IsZero() {
		// Outlive the access token: the refresh flow still reads it afterwards.
		c.Cookie(sessionCookie(tokenExpiryCookie, strconv.FormatInt(tokens.ExpiresAt.Unix(), 10), tokens.RefreshExpiresAt))
	}
	if tokens.RefreshToken != "" {
		c.Cookie(sessionCookie(refreshTokenCookie, tokens.RefreshToken, tokens.RefreshExpiresAt))
	}
}

func clearTokens(c fiber.Ctx) {
	clearCookie(c, accessTokenCookie)
	clearCookie(c, refreshTokenCookie)
	clearCookie(c, tokenExpiryCookie)
}
