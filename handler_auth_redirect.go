package main

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

func authRedirectHandler(c *fiber.Ctx) error {

	// Once the user approves authorization for your app, they will be sent to your redirect URI as indicated in the request.
	// 		We will add the following parameters as we make the call to your redirect URI:
	//			code: This is the code you will use in the next step to exchange for an access token.
	//			state: This is the optional parameter to prevent cross-site request forgery. Check to make sure it matches what was passed in the first step of the flow. If it does not, the exchange may have been subject to a cross-site request forgery attack.
	// A redirect URI such as https://www.example.com/oauth/pinterest/oauth_response/
	// 		will result in a callback request like: https://www.example.com/oauth/pinterest/oauth_response/?code={CODE}&state={YOUR_OPTIONAL_STRING}

	codeKey := c.Query("code")

	if len(codeKey) > 0 {
		log.Println("Code Key: " + codeKey)

		err := app.PinClient.FetchAccessToken(codeKey)

		if err != nil {
			return err
		}

		cookie := fiber.Cookie{
			Name:    "access_token",
			Value:   app.PinClient.AccessToken,
			Expires: time.Now().Add(365 * 24 * time.Hour),
		}

		c.Cookie(&cookie)

		log.Printf("Cookie: %s, value: %s", cookie.Name, cookie.Value)

		log.Println("Success! Go back home!")

		c.Redirect(app.PinClient.MainURL)
	}

	return nil
}
