package main

import (
	"log"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	pinterestControllers "github.com/iggyzuk/go-pinterest/controllers"
	pinterestModels "github.com/iggyzuk/go-pinterest/models"
	"github.com/iggyzuk/shuffle/controllers"
	"github.com/iggyzuk/shuffle/models"
)

var oauthURL = "https://api.pinterest.com/oauth/?response_type=code&redirect_uri=" + rootURL + "/redirect&client_id=" + clientID + "&scope=read_public,read_relationships"

var user *pinterestModels.User

func boardFromURL(boardURL string) string {
	u, _ := url.Parse(boardURL)
	board := u.Path
	board = strings.TrimPrefix(board, "/")
	board = strings.TrimSuffix(board, "/")
	return board
}

func fetchMyBoards(tmplData *models.TemplateData) {
	user, _ = client.Me.Fetch()

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
			URL:      boardFromURL(board.Url),
			PinCoint: board.Counts.Pins,
		})
	}

	time.Sleep(1000 * time.Millisecond)
}

func fetchFollowedBoards(tmplData *models.TemplateData) {
	optionals := pinterestControllers.MeFollowingBoardsFetchOptionals{}
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
			URL:      boardFromURL(board.Url),
			PinCoint: board.Counts.Pins,
		})
	}

	time.Sleep(1000 * time.Millisecond)
}

// renders page after passing some data to the HTML template
func indexHandler(c *fiber.Ctx) error {

	tmplData := models.TemplateData{
		OAuthURL:       oauthURL,
		Authenticated:  false,
		Boards:         []models.Board{},
		FollowedBoards: []models.Board{},
		Pins:           []models.Pin{},
	}

	accessTokenCookie := new(fiber.Cookie)
	accessTokenCookie.Name = "access_token"
	accessTokenCookie.Value = "access_token"
	accessTokenCookie.Expires = time.Now().Add(24 * time.Hour)

	c.Cookie(accessTokenCookie)

	if c.Cookies(accessTokenCookie.Name) == "" {
		log.Println("Missing Cookie")

	} else {
		log.Println("Cookie Exists")

		client = client.RegisterAccessToken(accessTokenCookie.Name)

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

			boardKeys := controllers.ParseBoards(c.OriginalURL())

			if len(boardKeys) > 0 {

				max := controllers.ParseMax(c.OriginalURL())

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

	return c.Render("layout", tmplData)
}
