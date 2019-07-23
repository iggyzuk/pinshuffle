package main

import (
	"html/template"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"sort"

	goctrl "github.com/carrot/go-pinterest/controllers"
	"iggyzuk.com/shuffle/controllers"
	"iggyzuk.com/shuffle/models"
)

var oauthURL = "https://api.pinterest.com/oauth/?response_type=code&redirect_uri=" + rootURL + "/redirect&client_id=" + clientID + "&scope=read_public,read_relationships"

func fetchMyBoards(tmplData *models.TemplateData) {
	boards, err := client.Me.Boards.Fetch()

	if err != nil {
		log.Println(err.Error())
		tmplData.Error = err.Error()
		return
	}

	// Sort boards by name
	sort.Slice(*boards, func(a, b int) bool {
		return (*boards)[a].Name < (*boards)[b].Name
	})

	// this fills up the board modal
	for _, board := range *boards {
		tmplData.Boards = append(tmplData.Boards, models.Board{
			Name:     board.Name,
			URL:      path.Base(board.Creator.Url) + "/" + path.Base(board.Url),
			PinCoint: board.Counts.Pins,
		})
	}
}

func fetchFollowedBoards(tmplData *models.TemplateData) {
	optionals := goctrl.MeFollowingBoardsFetchOptionals{}
	boards, _, err := client.Me.Following.Boards.Fetch(&optionals)

	if err != nil {
		log.Println(err.Error())
		tmplData.Error = err.Error()
		return
	}

	// Sort boards by name
	sort.Slice(*boards, func(a, b int) bool {
		return (*boards)[a].Name < (*boards)[b].Name
	})

	// this fills up the board modal
	for _, board := range *boards {
		tmplData.FollowedBoards = append(tmplData.FollowedBoards, models.Board{
			Name:     board.Name,
			URL:      path.Base(board.Creator.Url) + "/" + path.Base(board.Url),
			PinCoint: board.Counts.Pins,
		})
	}
}

// renders page after passing some data to the HTML template
func indexHandler(w http.ResponseWriter, req *http.Request) {

	tmplData := models.TemplateData{
		OAuthURL:       oauthURL,
		Authenticated:  false,
		Boards:         []models.Board{},
		FollowedBoards: []models.Board{},
		Pins:           []models.Pin{},
	}

	accessTokenCookie, err := req.Cookie("access_token")

	if err == http.ErrNoCookie {
		log.Println("Missing Cookie")

	} else {
		log.Println("Cookie Exists")

		client = client.RegisterAccessToken(accessTokenCookie.Value)

		_, err := client.Me.Fetch()

		if err != nil {
			log.Println(err.Error())
			tmplData.Error = err.Error()

		} else {
			log.Println("User Authenticated")

			tmplData.Authenticated = true

			fetchMyBoards(&tmplData)
			fetchFollowedBoards(&tmplData)

			tmplData.TotalBoards = len(tmplData.Boards) + len(tmplData.FollowedBoards)

			boardKeys := controllers.ParseBoards(req.URL)

			if len(boardKeys) > 0 {

				max := controllers.ParseMax(req.URL)

				controllers.Randomize()

				pins, err := controllers.GetBoardsPins(
					client,
					boardKeys,
					max,
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
