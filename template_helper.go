package main

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/ahmetb/go-linq/v3"
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
}

type TemplateBoard struct {
	Name     string
	Id       string
	PinCount int
}

type TemplatePin struct {
	ImageURL string
	PinURL   string
	Color    string
}

type TemplateUrlQuery struct {
	Max    int
	Boards []string
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
	}
}

func (tm *TemplateModel) Mock() {
	tm.OAuthURL = ""
	tm.Authenticated = true
	tm.Boards = []TemplateBoard{
		{Name: "Visual Style", Id: "visual-style", PinCount: 1275},
		{Name: "Ideas", Id: "ideas", PinCount: 947},
		{Name: "Concepts", Id: "concepts", PinCount: 802},
	}
	tm.Pins = []TemplatePin{
		{ImageURL: "https://iggyzuk.com/img/profile/iggy.jpg", PinURL: "#", Color: "#000000"},
		{ImageURL: "https://iggyzuk.com/projects/deadly-30/img/d30.gif", PinURL: "#", Color: "#000000"},
		{ImageURL: "https://iggyzuk.com/projects/kings/img/kings.png", PinURL: "#", Color: "#000000"},
		{ImageURL: "https://iggyzuk.com/projects/ninja-rampage/img/ninja.gif", PinURL: "#", Color: "#000000"},
		{ImageURL: "https://iggyzuk.com/projects/red-baron/img/red-baron.gif", PinURL: "#", Color: "#000000"},
		{ImageURL: "https://iggyzuk.com/projects/swords-and-forks/img/swords-and-forks-animated.gif", PinURL: "#", Color: "#000000"},
		{ImageURL: "https://iggyzuk.com/projects/simple-outline/img/simple-outline.jpg", PinURL: "#", Color: "#000000"},
		{ImageURL: "https://iggyzuk.com/projects/greed-wars/img/greed-wars-animated.gif", PinURL: "#", Color: "#000000"},
		{ImageURL: "https://iggyzuk.com/projects/game-engine/img/cover.png", PinURL: "#", Color: "#000000"},
		{ImageURL: "https://iggyzuk.com/projects/game-engine/img/books/game-coding.jpg", PinURL: "#", Color: "#000000"},
		{ImageURL: "https://iggyzuk.com/projects/game-engine/img/code.png", PinURL: "#", Color: "#000000"},
		{ImageURL: "https://iggyzuk.com/projects/forest-monster/img/design/depth.jpg", PinURL: "#", Color: "#000000"},
	}
	tm.Error = "Mock Error!"
	tm.Message = "Mock Message..."
	// tm.UrlQuery = &TemplateUrlQuery{}
}

func IsBoardSelected(id string) bool {
	return slices.Contains(randomizer.Boards, id)
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
		for _, b := range queryMap["b"] {

			valid := linq.From(tm.Boards).Where(func(board interface{}) bool {
				return board.(TemplateBoard).Id == b
			}).Any()

			if valid {
				tm.UrlQuery.Boards = append(tm.UrlQuery.Boards, b)
			}

		}
	}

	fmt.Printf("TemplateQuery: %+v", tm.UrlQuery)

	return nil
}
