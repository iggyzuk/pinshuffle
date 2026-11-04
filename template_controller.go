package main

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"

	"github.com/valyala/fasthttp"
)

const (
	defaultMaxPins         = 100
	minMaxPins             = 1
	maxMaxPins             = 10000
	defaultImageResolution = 1
	maxImageResolution     = 3
)

type TemplateController struct {
	Model TemplateModel
}

// TemplateModel is the main object we pass for templating HTML
type TemplateModel struct {
	Authenticated bool
	User          TemplateUser
	Boards        map[string]*TemplateBoard
	Pins          []TemplatePin
	Error         string
	Message       string
	UrlQuery      TemplateUrlQuery
	Loading       bool
	TaskId        string
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
	Width    int
	Height   int
	AltText  string
	Board    *TemplateBoard
}

type TemplateUrlQuery struct {
	Boards          []string
	Max             int
	ImageResolution int
}

func (query TemplateUrlQuery) GetImageResolution(images Images) Image {
	switch query.ImageResolution {
	case 0:
		return images.Res150x150
	case 2:
		return images.Res600x
	case 3:
		return images.Res1200x
	default:
		return images.Res400x300
	}
}

func NewTemplateController() *TemplateController {
	return &TemplateController{
		Model: TemplateModel{
			Boards: make(map[string]*TemplateBoard),
		},
	}
}

func (tc *TemplateController) AddBoard(board *TemplateBoard) {
	tc.Model.Boards[board.Id] = board
}

func (tc *TemplateController) AddPin(pin *Pin) {
	// The board can be gone by now, if it was deleted while the task was running.
	board, exists := tc.Model.Boards[pin.BoardId]
	if !exists {
		board = &TemplateBoard{}
	}

	image := tc.Model.UrlQuery.GetImageResolution(pin.Media.Images)

	tc.Model.Pins = append(tc.Model.Pins, TemplatePin{
		Id:       pin.Id,
		Name:     pin.Title,
		Color:    pin.DominantColor,
		ImageURL: image.Url,
		Width:    image.Width,
		Height:   image.Height,
		AltText:  pin.AltText,
		Board:    board,
	})
}

func (tc *TemplateController) Mock(uri *fasthttp.URI) {

	tc.Model.Authenticated = true

	tc.Model.User = TemplateUser{Name: "Iggy Zuk", IconURL: "https://iggyzuk.com/img/profile/iggy.jpg", URL: "#"}

	var clientBoards = make(map[string]Board)

	tc.MockBoard(clientBoards, "visits")
	tc.MockBoard(clientBoards, "insightful edge")
	tc.MockBoard(clientBoards, "novel clocks")
	tc.MockBoard(clientBoards, "misty rocks")
	tc.MockBoard(clientBoards, "network")
	tc.MockBoard(clientBoards, "beautiful long name")
	tc.MockBoard(clientBoards, "last power")
	tc.MockBoard(clientBoards, "essential")
	tc.MockBoard(clientBoards, "strikebreaker")
	tc.MockBoard(clientBoards, "punish reform")
	tc.MockBoard(clientBoards, "tourists")
	tc.MockBoard(clientBoards, "marine")
	tc.MockBoard(clientBoards, "heavy")
	tc.MockBoard(clientBoards, "computers")
	tc.MockBoard(clientBoards, "pipe trance")
	tc.MockBoard(clientBoards, "hammer")
	tc.MockBoard(clientBoards, "connection")
	tc.MockBoard(clientBoards, "fat")
	tc.MockBoard(clientBoards, "monarch")
	tc.MockBoard(clientBoards, "extreme zero")
	tc.MockBoard(clientBoards, "possibility")
	tc.MockBoard(clientBoards, "dressing")
	tc.MockBoard(clientBoards, "off")
	tc.MockBoard(clientBoards, "terminal")
	tc.MockBoard(clientBoards, "slam dunk")

	tc.ParseUrlQueries(uri, clientBoards)

	boardList := SortBoards(tc.Model.Boards)

	mockPins := []TemplatePin{
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

	// Repeat the pins up to max, to have enough of them to fill the page.
	for i := 0; i < tc.Model.UrlQuery.Max; i++ {
		pin := mockPins[i%len(mockPins)]
		pin.Id = fmt.Sprintf("#%d", i+1)
		tc.Model.Pins = append(tc.Model.Pins, pin)
	}

	tc.Model.Message = "Mock"
	tc.Model.Error = "This is an example error!"
}

func (tc *TemplateController) MockBoard(boards map[string]Board, id string) Board {
	board := Board{Id: id, Name: id}
	boards[id] = board
	return board
}

// Parses the URI for max pins, boards, image-size.
func (tc *TemplateController) ParseUrlQueries(uri *fasthttp.URI, clientBoards map[string]Board) error {
	queryString := string(uri.QueryString())

	queryMap, err := url.ParseQuery(queryString)
	if err != nil {
		return err
	}

	query := &tc.Model.UrlQuery

	query.Max = parseIntInRange(queryMap, "max", defaultMaxPins, minMaxPins, maxMaxPins)
	query.ImageResolution = parseIntInRange(queryMap, "res", defaultImageResolution, 0, maxImageResolution)

	for _, boardId := range queryMap["b"] {
		board, exists := clientBoards[boardId]
		if exists && !slices.Contains(query.Boards, board.Id) {
			query.Boards = append(query.Boards, board.Id)
		}
	}

	for _, board := range clientBoards {
		tc.AddBoard(&TemplateBoard{Id: board.Id, Name: board.Name})
	}

	return nil
}

// parseIntInRange reads a query value, limiting it to [low, high]. Anything that
// isn't a number gets the default.
func parseIntInRange(queryMap url.Values, key string, defaultValue, low, high int) int {
	value, err := strconv.Atoi(queryMap.Get(key))
	if err != nil {
		return defaultValue
	}
	return max(low, min(value, high))
}
