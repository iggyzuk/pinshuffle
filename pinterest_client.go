package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
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

type UserAccount struct {
	ProfileImage string `json:"profile_image"`
	WebsiteURL   string `json:"website_url"`
	Username     string `json:"username"`
}

type Boards struct {
	Items    []Board `json:"items"`
	Bookmark string  `json:"bookmark"`
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
	Id            string `json:"id"`
	Title         string `json:"title"`
	DominantColor string `json:"dominant_color"`
	Media         Media  `json:"media"`
	BoardId       string `json:"board_id"`
	AltText       string `json:"alt_text"`
}

type Media struct {
	Images Images `json:"images"`
}

type Images struct {
	Res150x150 Image `json:"150x150"`
	Res400x300 Image `json:"400x300"`
	Res600x    Image `json:"600x"`
	Res1200x   Image `json:"1200x"`
}

type Image struct {
	Url string `json:"url"`
}

func NewClient() *PinterestClient {

	return &PinterestClient{
		AppID:       os.Getenv("CLIENT_ID"),
		Secret:      os.Getenv("CLIENT_SECRET"),
		MainURL:     "https://pinshuffle.fly.dev",
		BaseURL:     "https://api.pinterest.com/v5",
		RedirectUri: "https://pinshuffle.fly.dev/redirect/",
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

func (client *PinterestClient) ExecuteRequest(endpoint string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, client.Endpoint(endpoint), nil)

	if err != nil {
		return nil, err
	}

	req.Header = client.GetHeader()

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 200 {

		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)

		if err != nil {
			return nil, err
		}

		return nil, errors.New("⚠️ " + string(b))
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (client *PinterestClient) FetchUserAccount() (*UserAccount, error) {

	bytes, err := client.ExecuteRequest("/user_account")

	if err != nil {
		return nil, err
	}

	var userAccount = new(UserAccount)
	unmarshalErr := json.Unmarshal(bytes, &userAccount)
	if unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return userAccount, nil
}

func (client *PinterestClient) FetchBoards() ([]Board, error) {

	var resultBoards []Board

	boards, err := client.FetchBoard("")
	if err != nil {
		return nil, err
	}

	resultBoards = append(resultBoards, boards.Items...)

	for len(boards.Bookmark) > 0 {
		boards, err = client.FetchBoard(boards.Bookmark)
		if err != nil {
			return nil, err
		}
		resultBoards = append(resultBoards, boards.Items...)
	}

	return resultBoards, nil
}

func (client *PinterestClient) FetchBoard(bookmark string) (*Boards, error) {

	url := "/boards/?page_size=100"
	if len(bookmark) > 0 {
		url += "&bookmark=" + bookmark
	}

	bytes, err := client.ExecuteRequest(url)
	if err != nil {
		return nil, err
	}

	var board = new(Boards)
	unmarshalErr := json.Unmarshal(bytes, &board)
	if unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return board, nil
}

func (client *PinterestClient) FetchAllPins(board Board) ([]Pin, error) {

	var resultingPins []Pin

	pins, err := client.FetchPage(board, "")
	if err != nil {
		return nil, err
	}

	resultingPins = append(resultingPins, pins.Items...)

	for len(pins.Bookmark) > 0 {
		pins, err = client.FetchPage(board, pins.Bookmark)
		if err != nil {
			return nil, err
		}
		resultingPins = append(resultingPins, pins.Items...)
	}

	return resultingPins, nil
}

func (client *PinterestClient) FetchPage(board Board, bookmark string) (*Pins, error) {

	url := "/boards/" + board.Id + "/pins?page_size=100"
	if len(bookmark) > 0 {
		url += "&bookmark=" + bookmark
	}

	bytes, err := client.ExecuteRequest(url)
	if err != nil {
		return nil, err
	}

	var pins = new(Pins)
	unmarshalErr := json.Unmarshal(bytes, &pins)
	if unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return pins, nil
}
