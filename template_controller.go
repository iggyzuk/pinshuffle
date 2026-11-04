package main

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/valyala/fasthttp"
	"golang.org/x/sync/errgroup"
)

const (
	// Between the ids of a board and a section in the "s" query.
	sectionSeparator = ":"

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
	Build         BuildInfo
}

type TemplateUser struct {
	Name    string
	IconURL string
	URL     string
}

type TemplateBoard struct {
	Name string
	Id   string
	// Only for boards with a selected section, the browser loads the rest on demand.
	Sections []TemplateSection
}

type TemplateSection struct {
	Id       string
	Name     string
	Selected bool
}

type TemplatePin struct {
	Id       string
	Name     string
	Color    string
	ImageURL string
	FullURL  string
	Width    int
	Height   int
	AltText  string
	Board    *TemplateBoard
	Section  string
}

// Source says where the pin is from, for showing it on the pin.
func (pin TemplatePin) Source() string {
	if pin.Section != "" {
		return pin.Board.Name + " › " + pin.Section
	}
	return pin.Board.Name
}

type TemplateUrlQuery struct {
	Boards          []string
	Sections        []PinSource
	Max             int
	ImageResolution int
}

// Sources is everything the user chose to shuffle.
func (query TemplateUrlQuery) Sources() []PinSource {
	sources := make([]PinSource, 0, len(query.Boards)+len(query.Sections))
	for _, boardId := range query.Boards {
		sources = append(sources, PinSource{BoardId: boardId})
	}
	return append(sources, query.Sections...)
}

// GetImageResolution picks the wanted size, or the closest one that exists:
// Pinterest doesn't always return all of them.
func (query TemplateUrlQuery) GetImageResolution(images Images) Image {
	switch query.ImageResolution {
	case 0:
		return firstAvailable(images.Res150x150, images.Res400x300, images.Res600x, images.Res1200x)
	case 2:
		return firstAvailable(images.Res600x, images.Res1200x, images.Res400x300, images.Res150x150)
	case 3:
		return largestImage(images)
	default:
		return firstAvailable(images.Res400x300, images.Res600x, images.Res1200x, images.Res150x150)
	}
}

func largestImage(images Images) Image {
	return firstAvailable(images.Res1200x, images.Res600x, images.Res400x300, images.Res150x150)
}

func firstAvailable(candidates ...Image) Image {
	for _, image := range candidates {
		if image.Url != "" {
			return image
		}
	}
	return Image{}
}

func NewTemplateController() *TemplateController {
	return &TemplateController{
		Model: TemplateModel{
			Boards: make(map[string]*TemplateBoard),
			Build:  build,
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

	// Nothing to show, it would only be a blank square.
	if image.Url == "" {
		return
	}

	section := ""
	for _, candidate := range board.Sections {
		if candidate.Id == pin.BoardSectionId {
			section = candidate.Name
		}
	}

	tc.Model.Pins = append(tc.Model.Pins, TemplatePin{
		Id:       pin.Id,
		Name:     pin.Title,
		Color:    pin.DominantColor,
		ImageURL: image.Url,
		FullURL:  largestImage(pin.Media.Images).Url,
		Width:    image.Width,
		Height:   image.Height,
		AltText:  pin.AltText,
		Board:    board,
		Section:  section,
	})
}

func (tc *TemplateController) Mock(uri *fasthttp.URI) {

	tc.Model.Authenticated = true

	tc.Model.User = TemplateUser{Name: "Iggy Zuk", IconURL: "https://iggyzuk.com/projects/kings/img/kings.png", URL: "#"}

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
	tc.LoadSections(context.Background(), func(_ context.Context, boardId string) ([]Section, error) {
		return mockSections(boardId), nil
	})

	tc.Model.Message = "Mock"
	tc.Model.Error = "This is an example error!"
}

// AddMockPins fills the page with up to max example pins.
func (tc *TemplateController) AddMockPins() {
	boardList := SortBoards(tc.Model.Boards)

	mockPins := []TemplatePin{
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
}

func (tc *TemplateController) MockBoard(boards map[string]Board, id string) Board {
	board := Board{Id: id, Name: id}
	boards[id] = board
	return board
}

func mockSections(boardId string) []Section {
	if boardId != "visits" && boardId != "marine" && boardId != "computers" {
		return []Section{}
	}
	return []Section{
		{Id: "s1", Name: "first steps"},
		{Id: "s2", Name: "reference"},
		{Id: "s3", Name: "a section with a rather long name"},
	}
}

// Parses the URI for max pins, boards, sections, image-size.
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

	for _, raw := range queryMap["s"] {
		boardId, sectionId, found := strings.Cut(raw, sectionSeparator)
		source := PinSource{BoardId: boardId, SectionId: sectionId}

		// Picking a whole board already includes all of its sections.
		_, boardExists := clientBoards[boardId]
		if found && boardExists && !slices.Contains(query.Boards, boardId) && !slices.Contains(query.Sections, source) {
			query.Sections = append(query.Sections, source)
		}
	}

	return nil
}

// LoadSections gets the sections of the boards that have some selected. This
// shows them in the board list, and drops selections that don't exist. The other
// boards stay unloaded: that would take a request per board on every page load.
func (tc *TemplateController) LoadSections(ctx context.Context, fetch func(context.Context, string) ([]Section, error)) error {
	query := &tc.Model.UrlQuery
	sectionsByBoard := make(map[string][]Section)

	boardIds := make([]string, 0, len(query.Sections))
	for _, source := range query.Sections {
		if !slices.Contains(boardIds, source.BoardId) {
			boardIds = append(boardIds, source.BoardId)
		}
	}

	var mu sync.Mutex
	var group errgroup.Group
	group.SetLimit(maxConcurrentBoards)

	for _, boardId := range boardIds {
		group.Go(func() error {
			sections, err := fetch(ctx, boardId)
			if err != nil {
				return err
			}

			mu.Lock()
			defer mu.Unlock()
			sectionsByBoard[boardId] = sections
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return err
	}

	selected := query.Sections
	query.Sections = nil

	for boardId, sections := range sectionsByBoard {
		board := tc.Model.Boards[boardId]

		for _, section := range sections {
			source := PinSource{BoardId: boardId, SectionId: section.Id}
			isSelected := slices.Contains(selected, source)

			board.Sections = append(board.Sections, TemplateSection{Id: section.Id, Name: section.Name, Selected: isSelected})

			if isSelected {
				query.Sections = append(query.Sections, source)
			}
		}
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
