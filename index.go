package main

import (
	"html/template"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"iggyzuk.com/shuffle/controllers"
	"iggyzuk.com/shuffle/models"
)

var oauthURL = "https://api.pinterest.com/oauth/?response_type=code&redirect_uri=" + rootURL + "/redirect&client_id=" + clientID + "&scope=read_public"

// renders page after passing some data to the HTML template
func indexHandler(w http.ResponseWriter, req *http.Request) {

	tmplData := models.TemplateData{
		OAuthURL:      oauthURL,
		Authenticated: false,
		Boards:        []models.Board{},
		Pins:          []models.Pin{},
	}

	accessTokenCookie, err := req.Cookie("access_token")

	if err == http.ErrNoCookie {
		log.Println("Missing Cookie")

	} else {
		log.Println("Cookie Exists")

		client = client.RegisterAccessToken(accessTokenCookie.Value)

		user, err := client.Me.Fetch()

		if err != nil {
			log.Println(err.Error())
			tmplData.Error = err.Error()

		} else {
			log.Println("User Authenticated")

			tmplData.Authenticated = true

			boards, err := client.Me.Boards.Fetch()

			if err != nil {
				log.Println(err.Error())
				tmplData.Error = err.Error()

			} else {

				// this fills up the board modal
				for _, board := range *boards {
					userSlashBoard := strings.ToLower(user.Username + "/" + path.Base(board.Url))
					tmplData.Boards = append(tmplData.Boards, models.Board{
						Name:     board.Name,
						URL:      rootURL + "/?b=" + userSlashBoard,
						PinCoint: board.Counts.Pins,
					})
				}
			}

			boardKeys := controllers.ParseBoards(req.URL)

			if len(boardKeys) > 0 {

				pins, err := controllers.GetBoardsPins(
					client,
					boardKeys,
					accessTokenCookie.Value,
				)

				if err != nil {
					tmplData.Error = err.Error()
				} else {
					tmplData.Pins = *pins
				}

			} else {
				tmplData.Message = "You can select your boards in the bottom right. You can also modify the URL directly: ?b=username/board"
			}

		}
	}

	// Build path to template
	tmplPath := filepath.Join(templatesDir, "layout.gohtml")
	// Load template from disk
	tmpl := template.Must(template.ParseFiles(tmplPath))

	tmpl.Execute(w, tmplData)
}
