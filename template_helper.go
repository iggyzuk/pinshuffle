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
	User          TemplateUser
	Boards        []TemplateBoard
	BoardMap      map[string]*TemplateBoard // for convinience.
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

func IsBoardSelected(boards []string, id string) bool {
	return slices.Contains(boards, id)
}

func NewTemplateModel(authUrl string) *TemplateModel {
	return &TemplateModel{
		OAuthURL:      authUrl,
		Authenticated: false,
		User:          TemplateUser{Name: "unknown", IconURL: "#", URL: "#"},
		Boards:        nil,
		BoardMap:      make(map[string]*TemplateBoard),
		Pins:          nil,
		Error:         "",
		Message:       "",
		UrlQuery:      TemplateUrlQuery{},
		ImageSize:     3,
	}
}

func (tm *TemplateModel) Mock(uri *fasthttp.URI, clientBoards map[string]*Board) {
	tm.OAuthURL = ""
	tm.Authenticated = true

	tm.User = TemplateUser{Name: "Iggy Zuk", IconURL: "https://iggyzuk.com/img/profile/iggy.jpg", URL: "#"}

	tm.ParseUrlQueries(uri, clientBoards)

	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Test Board"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Nice"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Amazing And Long Name"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Best Stuff"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Maps"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Another Long Long Board Name, Very Cool"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Generic"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Wallpapers"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Cool Topologies"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "A"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "AB"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABC"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCD"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDE"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEF"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFG"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFGH"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFGHI"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFGHIJ"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFGHIJK"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Test Board"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Nice"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Amazing And Long Name"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Best Stuff"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Maps"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Another Long Long Board Name, Very Cool"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Generic"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Wallpapers"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Cool Topologies"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "A"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "AB"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABC"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCD"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDE"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEF"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFG"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFGH"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFGHI"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFGHIJ"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFGHIJK"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Test Board"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Nice"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Amazing And Long Name"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Best Stuff"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Maps"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Another Long Long Board Name, Very Cool"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Generic"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Wallpapers"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "Cool Topologies"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "A"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "AB"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABC"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCD"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDE"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEF"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFG"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFGH"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFGHI"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFGHIJ"})
	tm.Boards = append(tm.Boards, TemplateBoard{Name: "ABCDEFGHIJK"})

	tm.Pins = []TemplatePin{
		{Id: "#1", Name: "Iggy", Color: "#000000", ImageURL: "https://iggyzuk.com/img/profile/iggy.jpg", Board: &tm.Boards[0], AltText: "Iggy Zuky"},
		{Id: "#2", Name: "Deadly 30", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/deadly-30/img/d30.gif", Board: &tm.Boards[0]},
		{Id: "#3", Name: "Kings", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/kings/img/kings.png", Board: &tm.Boards[0]},
		{Id: "#4", Name: "Ninja Rampage", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/ninja-rampage/img/ninja.gif", Board: &tm.Boards[0]},
		{Id: "#5", Name: "Red Baron", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/red-baron/img/red-baron.gif", Board: &tm.Boards[1]},
		{Id: "#6", Name: "Forks & Swords", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/swords-and-forks/img/swords-and-forks-animated.gif", Board: &tm.Boards[1]},
		{Id: "#7", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/simple-outline/img/simple-outline.jpg", Board: &TemplateBoard{}},
		{Id: "#8", Name: "Greed Wars", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/greed-wars/img/greed-wars-animated.gif", Board: &tm.Boards[1]},
		{Id: "#9", Name: "Custom Engine (Mario)", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/game-engine/img/cover.png", Board: &tm.Boards[2]},
		{Id: "#10", Name: "Game Coding Complete", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/game-engine/img/books/game-coding.jpg", Board: &tm.Boards[2]},
		{Id: "#11", Name: "Code", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/game-engine/img/code.png", Board: &tm.Boards[2]},
		{Id: "#12", Name: "Forest Monster", Color: "#000000", ImageURL: "https://iggyzuk.com/projects/forest-monster/img/design/depth.jpg", Board: &tm.Boards[2]},
	}

	tm.Message = "Mock"
}

func (tm *TemplateModel) ParseUrlQueries(uri *fasthttp.URI, clientBoards map[string]*Board) error {
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

		newTemplateBoard := TemplateBoard{
			Id:   val.Id,
			Name: val.Name,
		}

		tm.Boards = append(tm.Boards, newTemplateBoard)

		// Store board pointer in our convinience map.
		tm.BoardMap[newTemplateBoard.Id] = &newTemplateBoard

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
