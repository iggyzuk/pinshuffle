package main

import (
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// httpsRedirect redirects http requests to https
func httpsRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(
		w, r,
		"https://"+r.Host+r.URL.String(),
		http.StatusMovedPermanently,
	)
}

// pinterestRedirectHandler redirects with OAuth2 code
func pinterestRedirectHandler(c *fiber.Ctx) error {
	// codeKey := req.FormValue("code")

	var codeKey string = "temp"

	if len(codeKey) > 0 {
		log.Println("Access Code: " + codeKey)

		// accessToken, err := client.OAuth.Token.Create(
		// 	clientID,
		// 	clientSecret,
		// 	codeKey,
		// )

		// if err != nil {
		// 	log.Println("Something went wrong with the redirect code")
		// 	log.Println(err)

		// 	w.WriteHeader(http.StatusUnauthorized)
		// 	w.Header().Set("Content-Type", "application/json")

		// 	io.WriteString(w, err.Error())
		// 	return
		// }

		// client = client.RegisterAccessToken(accessToken.AccessToken)
		// log.Println("Access Token: " + accessToken.AccessToken)

		// cookie := http.Cookie{
		// 	Name:    "access_token",
		// 	Value:   accessToken.AccessToken,
		// 	Expires: time.Now().Add(365 * 24 * time.Hour),
		// }

		// http.SetCookie(w, &cookie)

		log.Println("Success. Go to index!")

		// http.Redirect(w, req, rootURL, http.StatusMovedPermanently)
	}

	return c.SendString("Not Implemented!")
}
