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
	// Name     string
	// URL      string
	// PinCount int
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
	tm.Pins = []TemplatePin{}
	tm.Error = "Manual Error"
	tm.Message = "Manual Message"
	tm.UrlQuery = &TemplateUrlQuery{}
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
