package main

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"

	"github.com/valyala/fasthttp"
	"golang.org/x/exp/slices"
)

// TemplateModel is the main object we pass for templating HTML
type TemplateModel struct {
	OAuthURL      string
	Authenticated bool
	Boards        []TemplateBoard
	Pins          []TemplatePin
	Error         string
	Message       string
	UrlQuery      *TemplateUrlQuery
	ImageSize     int
}

type TemplateBoard struct {
	Name string
	Id   string
}

type TemplatePin struct {
	Name     string
	Color    string
	ImageURL string
	Id       string
}

type TemplateUrlQuery struct {
	Boards          []string
	Max             int
	ImageResolution int
}

func IsBoardSelected(id string) bool {
	return slices.Contains(tm.UrlQuery.Boards, id)
}

func NewTemplateModel(authUrl string) *TemplateModel {
	return &TemplateModel{
		OAuthURL:      authUrl,
		Authenticated: false,
		Boards:        nil,
		Pins:          nil,
		Error:         "",
		Message:       "",
		UrlQuery:      &TemplateUrlQuery{},
		ImageSize:     3,
	}
}

func (tm *TemplateModel) Mock() {
	tm.OAuthURL = ""
	tm.Authenticated = true

	clientBoards["visual-style"] = &Board{Id: "visual-style", Name: "Visual Style"}
	clientBoards["ideas"] = &Board{Id: "ideas", Name: "Ideas"}
	clientBoards["concepts"] = &Board{Id: "concepts", Name: "Concepts"}

	// ### You can override template board directly:

	// tm.Boards = []TemplateBoard{
	// 	{Name: "Visual Style", Id: "visual-style"},
	// 	{Name: "Ideas", Id: "ideas"},
	// 	{Name: "Concepts", Id: "concepts"},
	// }

	tm.Pins = []TemplatePin{
		{Id: "#1", Name: "Iggy", Color: "#000000", ImageURL: "https://iggyzuk.com/img/profile/iggy.jpg"},
		{Id: "#2", Name: "Deadly 30", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/deadly-30/img/d30.gif"},
		{Id: "#3", Name: "Kings", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/kings/img/kings.png"},
		{Id: "#4", Name: "Ninja Rampage", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/ninja-rampage/img/ninja.gif"},
		{Id: "#5", Name: "Red Baron", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/red-baron/img/red-baron.gif"},
		{Id: "#6", Name: "Forks & Swords", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/swords-and-forks/img/swords-and-forks-animated.gif"},
		{Id: "#7", Name: "Simple Outline", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/simple-outline/img/simple-outline.jpg"},
		{Id: "#8", Name: "Greed Wars", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/greed-wars/img/greed-wars-animated.gif"},
		{Id: "#9", Name: "Custom Engine (Mario)", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/game-engine/img/cover.png"},
		{Id: "#10", Name: "Game Coding Complete", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/game-engine/img/books/game-coding.jpg"},
		{Id: "#11", Name: "Code", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/game-engine/img/code.png"},
		{Id: "#12", Name: "Forest Monster", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/forest-monster/img/design/depth.jpg"},
	}
	// tm.Error = "Mock Error!"
	// tm.Message = "Mock Message..."
	// tm.UrlQuery = &TemplateUrlQuery{}
}

func (tm *TemplateModel) ParseUrlQueries(uri *fasthttp.URI) error {
	queryString := string(uri.QueryString())
	fmt.Println(queryString)

	queryMap, err := url.ParseQuery(queryString)
	if err != nil {
		return err
	}
	fmt.Println(queryMap)

	// Max.
	tm.UrlQuery.Max = 100

	if queryMap["max"] != nil {
		maxString := queryMap["max"][0]
		maxInt, err := strconv.Atoi(maxString)

		if err != nil {
			fmt.Println(err)
		} else {
			tm.UrlQuery.Max = maxInt
		}
	}

	// Boards.
	if queryMap["b"] != nil {
		for _, boardId := range queryMap["b"] {

			board, exists := clientBoards[boardId]
			if exists {
				tm.UrlQuery.Boards = append(tm.UrlQuery.Boards, board.Id)
			}
		}
	}

	fmt.Printf("Query: %+v \n", tm.UrlQuery)

	for _, val := range clientBoards {
		tm.Boards = append(tm.Boards, TemplateBoard{
			Id:   val.Id,
			Name: val.Name,
		})

		fmt.Printf("Board: %+v \n", val)
	}

	sort.SliceStable(tm.Boards, func(i, j int) bool {
		return tm.Boards[i].Name < tm.Boards[j].Name
	})

	// Image size.
	tm.UrlQuery.ImageResolution = 1

	if queryMap["res"] != nil {
		maxString := queryMap["res"][0]
		imgResInt, err := strconv.Atoi(maxString)

		if err != nil {
			fmt.Println(err)
		} else {
			tm.UrlQuery.ImageResolution = imgResInt
		}
	}

	return nil
}

func GetImageResolution(imgRes int, images Images) Image {
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
