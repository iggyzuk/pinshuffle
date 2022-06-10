package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type PinterestClient struct {
	AppID              string
	Secret             string
	MainURL            string
	BaseURL            string
	RedirectUri        string
	Scopes             string
	HttpClient         *http.Client
	DefaultContentType string
	AccessToken        string
}

type AccessTokenResponse struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	ResponseType          string `json:"response_type"`
	TokenType             string `json:"token_type"`
	ExpiresIn             int    `json:"expires_in"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	Scope                 string `json:"scope"`
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
		MainURL:     "https://pinshuffle.herokuapp.com/",
		BaseURL:     "https://api.pinterest.com/v5",
		RedirectUri: "https://pinshuffle.herokuapp.com/redirect/",
		Scopes:      "user_accounts:read,catalogs:read,boards:read,boards:read_secret,pins:read,pins:read_secret",
		HttpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		DefaultContentType: "application/json",
	}
}

func (client *PinterestClient) GetAuthUri() string {
	return "https://www.pinterest.com/oauth/?&client_id=" + client.AppID + "&redirect_uri=" + client.RedirectUri + "&response_type=code" + "&scope=" + client.Scopes
}

func (client *PinterestClient) FetchAccessToken(codeKey string) error {

	// Once you have received the code to your redirect URI,
	// you can exchange it for an access token by making a POST request
	// to our access token endpoint with a request body and content type of
	// application/x-www-form-urlencoded: https://api.pinterest.com/v5/oauth/token

	// curl -X POST https://api.pinterest.com/v5/oauth/token
	// --header 'Authorization: Basic {base64 encoded string made of client_id:client_secret}'
	// --header 'Content-Type: application/x-www-form-urlencoded'
	// --data-urlencode 'grant_type=authorization_code'
	// --data-urlencode 'code={YOUR_CODE}'
	// --data-urlencode 'redirect_uri=http://localhost/'

	data := url.Values{}
	data.Add("grant_type", "authorization_code")
	data.Add("redirect_uri", client.MainURL)
	data.Add("code", codeKey)

	request, err := http.NewRequest(
		http.MethodPost,
		"https://api.pinterest.com/v5/oauth/token",
		strings.NewReader(data.Encode()),
	)

	if err != nil {
		return err
	}

	request.SetBasicAuth(client.AppID, client.Secret)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.HttpClient.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	fmt.Printf("Response: %s\n", string(responseBytes))

	var accessTokenData AccessTokenResponse
	unmarshalErr := json.Unmarshal(responseBytes, &accessTokenData)
	if unmarshalErr != nil {
		return unmarshalErr
	}

	fmt.Printf("Access Token: %+v", accessTokenData)

	client.AccessToken = accessTokenData.AccessToken

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
	req, err := http.NewRequest(http.MethodGet, client.Endpoint(endpoint), nil)

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
