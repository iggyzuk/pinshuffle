package main

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"

	"github.com/valyala/fasthttp"
	"golang.org/x/exp/maps"
)

type TemplateController struct {
	Model TemplateModel
}

// TemplateModel is the main object we pass for templating HTML
type TemplateModel struct {
	OAuthURL      string
	Authenticated bool
	User          TemplateUser
	Boards        map[string]*TemplateBoard
	Pins          []TemplatePin
	Error         string
	Message       string
	UrlQuery      TemplateUrlQuery
	ImageSize     int
}

type TemplateUser struct {
	Name    string
	IconURL string
	URL     string
}

type TemplateBoard struct {
	Name string
	Id   string
}

type TemplatePin struct {
	Id       string
	Name     string
	Color    string
	ImageURL string
	AltText  string
	Board    *TemplateBoard
}

type TemplateUrlQuery struct {
	Boards          []string
	Max             int
	ImageResolution int
}

func NewTemplateController(authUrl string) *TemplateController {
	return &TemplateController{
		Model: TemplateModel{
			OAuthURL:      authUrl,
			Authenticated: false,
			User:          TemplateUser{Name: "unknown", IconURL: "#", URL: "#"},
			Boards:        make(map[string]*TemplateBoard),
			Pins:          nil,
			Error:         "",
			Message:       "",
			UrlQuery:      TemplateUrlQuery{},
			ImageSize:     3,
		},
	}
}

func (tc *TemplateController) AddBoard(board *TemplateBoard) {
	tc.Model.Boards[board.Id] = board
}

func (tc *TemplateController) GetBoards() []*TemplateBoard {
	return maps.Values(tc.Model.Boards)
}

func (tc *TemplateController) GetBoardsSorted() []*TemplateBoard {

	boardList := tc.GetBoards()

	sort.SliceStable(boardList, func(i, j int) bool {
		return boardList[i].Name < boardList[j].Name
	})

	return boardList
}

func (tc *TemplateController) GetImageResolution(images Images) Image {
	imgRes := tc.Model.UrlQuery.ImageResolution

	if imgRes == 0 {
		return images.Small
	}
	if imgRes == 1 {
		return images.Medium
	}
	if imgRes == 2 {
		return images.Huge
	}
	if imgRes == 3 {
		return images.Huge
	}
	if imgRes == 4 {
		return images.Original
	}
	return images.Medium
}

func (tc *TemplateController) AddPin(pin *Pin) {
	tc.Model.Pins = append(tc.Model.Pins, TemplatePin{
		Id:       pin.Id,
		Name:     pin.Title,
		Color:    pin.DominantColor,
		ImageURL: tc.GetImageResolution(pin.Media.Images).Url,
		AltText:  pin.AltText,
		Board:    tc.Model.Boards[pin.BoardId],
	})
}

func (tc *TemplateController) Mock(uri *fasthttp.URI) {

	tc.Model.OAuthURL = ""
	tc.Model.Authenticated = true

	tc.Model.User = TemplateUser{Name: "Iggy Zuk", IconURL: "https://iggyzuk.com/img/profile/iggy.jpg", URL: "#"}

	var clientBoards = make(map[string]*Board)

	tc.MockBoard(clientBoards, "A")
	tc.MockBoard(clientBoards, "AB")
	tc.MockBoard(clientBoards, "ABC")
	tc.MockBoard(clientBoards, "ABCD")
	tc.MockBoard(clientBoards, "ABCDE")
	tc.MockBoard(clientBoards, "ABCDEF")
	tc.MockBoard(clientBoards, "ABCDEFG")
	tc.MockBoard(clientBoards, "ABCDEFGHI")
	tc.MockBoard(clientBoards, "ABCDEFGHIJ")
	tc.MockBoard(clientBoards, "ABCDEFGHIJK")
	tc.MockBoard(clientBoards, "1")
	tc.MockBoard(clientBoards, "12")
	tc.MockBoard(clientBoards, "123")
	tc.MockBoard(clientBoards, "1234")
	tc.MockBoard(clientBoards, "12345")
	tc.MockBoard(clientBoards, "123456")
	tc.MockBoard(clientBoards, "1234567")
	tc.MockBoard(clientBoards, "12345678")
	tc.MockBoard(clientBoards, "123456789")
	tc.MockBoard(clientBoards, "1234567890")
	tc.MockBoard(clientBoards, "Visual Style")
	tc.MockBoard(clientBoards, "Game Ideas")
	tc.MockBoard(clientBoards, "Topologies")
	tc.MockBoard(clientBoards, "2D Animations")
	tc.MockBoard(clientBoards, "Animals")

	tc.ParseUrlQueries(uri, clientBoards)

	boardList := tc.GetBoardsSorted()

	tc.Model.Pins = []TemplatePin{
		{Id: "#1", Name: "Iggy", Color: "#000000", ImageURL: "https://iggyzuk.com/img/profile/iggy.jpg", Board: boardList[0], AltText: "Iggy Zuky"},
		{Id: "#2", Name: "Deadly 30", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/deadly-30/img/d30.gif", Board: boardList[0]},
		{Id: "#3", Name: "Kings", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/kings/img/kings.png", Board: boardList[0]},
		{Id: "#4", Name: "Ninja Rampage", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/ninja-rampage/img/ninja.gif", Board: boardList[0]},
		{Id: "#5", Name: "Red Baron", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/red-baron/img/red-baron.gif", Board: boardList[1]},
		{Id: "#6", Name: "Forks & Swords", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/swords-and-forks/img/swords-and-forks-animated.gif", Board: boardList[1]},
		{Id: "#7", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/simple-outline/img/simple-outline.jpg", Board: &TemplateBoard{}},
		{Id: "#8", Name: "Greed Wars", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/greed-wars/img/greed-wars-animated.gif", Board: boardList[1]},
		{Id: "#9", Name: "Custom Engine (Mario)", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/game-engine/img/cover.png", Board: boardList[2]},
		{Id: "#10", Name: "Game Coding Complete", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/game-engine/img/books/game-coding.jpg", Board: boardList[2]},
		{Id: "#11", Name: "Code", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/game-engine/img/code.png", Board: boardList[2]},
		{Id: "#12", Name: "Forest Monster", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/forest-monster/img/design/depth.jpg", Board: boardList[2]},
	}

	tc.Model.Message = "Mock"
}

func (tc *TemplateController) MockBoard(boards map[string]*Board, id string) *Board {
	board := &Board{Id: id, Name: id}
	boards[id] = board
	return board
}

func (tc *TemplateController) ParseUrlQueries(uri *fasthttp.URI, clientBoards map[string]*Board) error {
	queryString := string(uri.QueryString())
	fmt.Println(queryString)

	queryMap, err := url.ParseQuery(queryString)
	if err != nil {
		return err
	}
	fmt.Println(queryMap)

	// Max.
	tc.Model.UrlQuery.Max = 100

	if queryMap["max"] != nil {
		maxString := queryMap["max"][0]
		maxInt, err := strconv.Atoi(maxString)

		if err != nil {
			fmt.Println(err)
		} else {
			tc.Model.UrlQuery.Max = maxInt
		}
	}

	// Boards.
	if queryMap["b"] != nil {
		for _, boardId := range queryMap["b"] {

			board, exists := clientBoards[boardId]
			if exists {
				tc.Model.UrlQuery.Boards = append(tc.Model.UrlQuery.Boards, board.Id)
			}
		}
	}

	fmt.Printf("Query: %+v \n", tc.Model.UrlQuery)

	for _, val := range clientBoards {

		newTemplateBoard := &TemplateBoard{
			Id:   val.Id,
			Name: val.Name,
		}

		tc.AddBoard(newTemplateBoard)

		fmt.Printf("Board: %+v \n", val)
	}

	// Image size.
	tc.Model.UrlQuery.ImageResolution = 1

	if queryMap["res"] != nil {
		maxString := queryMap["res"][0]
		imgResInt, err := strconv.Atoi(maxString)

		if err != nil {
			fmt.Println(err)
		} else {
			tc.Model.UrlQuery.ImageResolution = imgResInt
		}
	}

	return nil
}
