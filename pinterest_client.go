package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
)

type PinterestClient struct {
	AppID              string
	Secret             string
	BaseURL            string
	RedirectUri        string
	Scopes             string
	HttpClient         *http.Client
	DefaultContentType string
	AccessToken        string
}

type AuthModel struct {
	AccessToken string `json:"access_token"`
}

type Boards struct {
	Items []Board `json:"items"`
}

type Board struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Pins struct {
	Items    []Pin  `json:"items"`
	Bookmark string `json:"bookmark"`
}

type Pin struct {
	Id    string `json:"id"`
	Title string `json:"title"`
	Color string `json:"dominant_color"`
	Media Media  `json:"media"`
}

type Media struct {
	Images Images `json:"images"`
}

type Images struct {
	Small    Image `json:"150x150"`
	Medium   Image `json:"400x300"`
	Large    Image `json:"600x"`
	Huge     Image `json:"1200x"`
	Original Image `json:"originals"`
}

type Image struct {
	Url string `json:"url"`
}

func NewClient(id string, secret string) *PinterestClient {

	return &PinterestClient{
		AppID:       "1478247",
		Secret:      "f40d064f73aaeabe98596eb9b409162742d36b69",
		BaseURL:     "https://api.pinterest.com/v5",
		RedirectUri: "https://pinshuffle.herokuapp.com/redirect/",
		Scopes:      "boards:read,pins:read",
		HttpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		DefaultContentType: "application/json",
	}
}

func (client *PinterestClient) GetAuthUri() string {
	return "https://www.pinterest.com/oauth/?&client_id=" + client.AppID + "&redirect_uri=" + client.RedirectUri + "&response_type=code" + "&scope=" + client.Scopes
}

func (client *PinterestClient) FetchAuthToken(codeKey string) error {

	// Once you have received the code to your redirect URI,
	// you can exchange it for an access token by making a POST request
	// to our access token endpoint with a request body and content type of
	// application/x-www-form-urlencoded: https://api.pinterest.com/v5/oauth/token

	body, _ := json.Marshal(map[string]string{
		"code":         codeKey,
		"redirect_uri": client.RedirectUri,
		"grant_type":   "authorization_code",
	})

	responseBody := bytes.NewBuffer(body)

	req, err := http.NewRequest("POST", "https://api.pinterest.com/v5/oauth/token", responseBody)

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req.SetBasicAuth(client.AppID, client.Secret)

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		return err
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var authModel AuthModel
	unmarshalErr := json.Unmarshal(bytes, &authModel)
	if unmarshalErr != nil {
		return unmarshalErr
	}

	client.AccessToken = authModel.AccessToken

	// Save cookie with auth token.
	accessTokenCookie := new(fiber.Cookie)
	accessTokenCookie.Name = "access_token"
	accessTokenCookie.Value = client.AccessToken
	accessTokenCookie.Expires = time.Now().Add(24 * time.Hour)

	return nil
}

func (client *PinterestClient) Endpoint(ext string) string {
	return client.BaseURL + ext
}

func (client *PinterestClient) GetHeader() http.Header {
	return http.Header{
		"Content-Type":  {client.DefaultContentType},
		"Authorization": {"Bearer " + client.AccessToken},
	}
}

func (client *PinterestClient) ExecuteRequest(endpoint string) []byte {
	req, err := http.NewRequest("GET", client.Endpoint(endpoint), nil)

	if err != nil {
		log.Fatal(err)
	}

	req.Header = client.GetHeader()

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return bytes
}

func (client *PinterestClient) FetchBoards() Boards {

	bytes := client.ExecuteRequest("/boards?page_size=100")

	var boards Boards
	unmarshalErr := json.Unmarshal(bytes, &boards)
	if unmarshalErr != nil {
		log.Fatal(unmarshalErr)
	}

	return boards
}

func (client *PinterestClient) FetchAllPins(board *Board) []Pin {

	var resultingPins []Pin

	pins := client.FetchPage(board, "")

	resultingPins = append(resultingPins, pins.Items...)

	for len(pins.Bookmark) > 0 {
		pins = client.FetchPage(board, pins.Bookmark)
		resultingPins = append(resultingPins, pins.Items...)
	}

	return resultingPins
}

func (client *PinterestClient) FetchPage(board *Board, bookmark string) Pins {

	url := "/boards/" + board.Id + "/pins?page_size=100"
	if len(bookmark) > 0 {
		url += "&bookmark=" + bookmark
	}

	bytes := client.ExecuteRequest(url)

	var pins Pins
	unmarshalErr := json.Unmarshal(bytes, &pins)
	if unmarshalErr != nil {
		log.Fatal(unmarshalErr)
	}

	return pins
}
