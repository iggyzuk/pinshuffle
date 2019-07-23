package models

// Pin is the model data that we use in template HTML
type Pin struct {
	ImageURL string
	PinURL   string
	Color    string
}

// Board is the model data that we use in template HTML
type Board struct {
	Name     string
	URL      string
	PinCoint int32
}

// TemplateData is the main object we pass for templating HTML
type TemplateData struct {
	OAuthURL       string
	Authenticated  bool
	Error          string
	Message        string
	Pins           []Pin
	Boards         []Board
	FollowedBoards []Board
	TotalBoards    int
}
