package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	defaultMainURL = "https://pinshuffle.fly.dev"

	maxRequestAttempts = 3
	maxRetryWait       = 30 * time.Second

	// Used when Pinterest omits the refresh token's lifetime.
	defaultRefreshTokenLifetime = 60 * 24 * time.Hour
)

// ErrUnauthorized means Pinterest rejected a token: the access token for API
// calls, or the refresh token for a refresh.
var ErrUnauthorized = errors.New("pinterest rejected the token")

// APIError is a non-200 response from Pinterest.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("⚠️ Pinterest error %d: %s", e.Status, e.Message)
}

func (e *APIError) Is(target error) bool {
	return target == ErrUnauthorized && e.Status == http.StatusUnauthorized
}

// newAPIError prefers the "message" field of Pinterest's JSON error body.
func newAPIError(status int, body []byte) *APIError {
	var parsed struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Message != "" {
		return &APIError{Status: status, Message: parsed.Message}
	}
	return &APIError{Status: status, Message: strings.TrimSpace(string(body))}
}

// PinterestClient holds the app's configuration. It is shared between users, so
// it must never carry a user's token: use WithToken to get a per-user copy.
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
	RetryDelay         time.Duration

	// Shared by copies, so concurrent requests holding the same refresh token
	// (two open tabs) only spend it once.
	refreshGroup *singleflight.Group
}

// tokenResponse is the body of POST /oauth/token for either grant.
type tokenResponse struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
	RefreshTokenExpiresAt int64  `json:"refresh_token_expires_at"`
}

type UserAccount struct {
	ProfileImage string `json:"profile_image"`
	WebsiteURL   string `json:"website_url"`
	Username     string `json:"username"`
}

type Board struct {
	Id   string `json:"id"`
	Name string `json:"name"`
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
	Url    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// page is one page of any bookmark-paginated Pinterest list.
type page[T any] struct {
	Items    []T    `json:"items"`
	Bookmark string `json:"bookmark"`
}

func NewClient() *PinterestClient {
	// BASE_URL lets you run a real OAuth flow somewhere other than production.
	// It has to be registered as a redirect URI with Pinterest.
	mainURL := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	if mainURL == "" {
		mainURL = defaultMainURL
	}

	return &PinterestClient{
		AppID:       os.Getenv("CLIENT_ID"),
		Secret:      os.Getenv("CLIENT_SECRET"),
		MainURL:     mainURL,
		BaseURL:     "https://api.pinterest.com/v5",
		RedirectUri: mainURL + "/redirect/",
		Scopes:      "user_accounts:read,catalogs:read,boards:read,boards:read_secret,pins:read,pins:read_secret",
		HttpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		DefaultContentType: "application/json",
		RetryDelay:         time.Second,
		refreshGroup:       new(singleflight.Group),
	}
}

// WithToken returns a copy of the client that makes requests as one user.
func (client *PinterestClient) WithToken(accessToken string) *PinterestClient {
	userClient := *client
	userClient.AccessToken = accessToken
	return &userClient
}

func (client *PinterestClient) GetAuthUri(state string) string {
	query := url.Values{}
	query.Set("client_id", client.AppID)
	query.Set("redirect_uri", client.RedirectUri)
	query.Set("response_type", "code")
	query.Set("scope", client.Scopes)
	query.Set("state", state)
	return "https://www.pinterest.com/oauth/?" + query.Encode()
}

// FetchAccessToken exchanges the OAuth code for the user's first set of tokens.
func (client *PinterestClient) FetchAccessToken(ctx context.Context, codeKey string) (Tokens, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", client.RedirectUri)
	data.Set("code", codeKey)
	// Apps created before 2025-09-25 need this to get a refresh token that can
	// be renewed. Pinterest ignores it for newer apps.
	data.Set("continuous_refresh", "true")

	return client.requestTokens(ctx, data, Tokens{})
}

// RefreshTokens trades the refresh token for a new access token. A rejected
// refresh token is reported as ErrUnauthorized.
func (client *PinterestClient) RefreshTokens(ctx context.Context, current Tokens) (Tokens, error) {
	result, err, _ := client.refreshGroup.Do(current.RefreshToken, func() (any, error) {
		data := url.Values{}
		data.Set("grant_type", "refresh_token")
		data.Set("refresh_token", current.RefreshToken)
		return client.requestTokens(ctx, data, current)
	})
	if err != nil {
		return Tokens{}, err
	}
	return result.(Tokens), nil
}

// requestTokens calls the token endpoint. Anything the response leaves out is
// carried over from previous.
func (client *PinterestClient) requestTokens(ctx context.Context, data url.Values, previous Tokens) (Tokens, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		client.Endpoint("/oauth/token"),
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return Tokens{}, err
	}

	request.SetBasicAuth(client.AppID, client.Secret)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.HttpClient.Do(request)
	if err != nil {
		return Tokens{}, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Tokens{}, err
	}

	if response.StatusCode != http.StatusOK {
		apiErr := newAPIError(response.StatusCode, body)
		if response.StatusCode >= 400 && response.StatusCode < 500 && response.StatusCode != http.StatusTooManyRequests {
			return Tokens{}, fmt.Errorf("%w: %w", ErrUnauthorized, apiErr)
		}
		return Tokens{}, apiErr
	}

	var parsed tokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Tokens{}, err
	}
	if parsed.AccessToken == "" {
		return Tokens{}, errors.New("pinterest token response has no access token")
	}

	now := time.Now()
	tokens := Tokens{AccessToken: parsed.AccessToken, RefreshToken: previous.RefreshToken, RefreshExpiresAt: previous.RefreshExpiresAt}

	if parsed.ExpiresIn > 0 {
		tokens.ExpiresAt = now.Add(time.Duration(parsed.ExpiresIn) * time.Second)
	}
	if parsed.RefreshToken != "" {
		tokens.RefreshToken = parsed.RefreshToken
		switch {
		case parsed.RefreshTokenExpiresAt > 0:
			tokens.RefreshExpiresAt = time.Unix(parsed.RefreshTokenExpiresAt, 0)
		case parsed.RefreshTokenExpiresIn > 0:
			tokens.RefreshExpiresAt = now.Add(time.Duration(parsed.RefreshTokenExpiresIn) * time.Second)
		}
	}
	if tokens.RefreshToken != "" && tokens.RefreshExpiresAt.IsZero() {
		tokens.RefreshExpiresAt = now.Add(defaultRefreshTokenLifetime)
	}

	return tokens, nil
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

// ExecuteRequest GETs an API endpoint, retrying rate limits and transient failures.
func (client *PinterestClient) ExecuteRequest(ctx context.Context, endpoint string) ([]byte, error) {
	for attempt := 1; ; attempt++ {
		body, wait, err := client.get(ctx, endpoint)

		retryable := wait >= 0 && ctx.Err() == nil && attempt < maxRequestAttempts
		if !retryable {
			return body, err
		}

		if wait == 0 {
			wait = client.RetryDelay << (attempt - 1)
		}
		log.Printf("Retrying Pinterest request (attempt %d) in %s: %v", attempt, wait, err)

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// get makes one request. The returned wait is how long to pause before retrying:
// 0 for "use the default backoff", negative if the request should not be retried.
func (client *PinterestClient) get(ctx context.Context, endpoint string) ([]byte, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.Endpoint(endpoint), nil)
	if err != nil {
		return nil, -1, err
	}

	req.Header = client.GetHeader()

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	switch {
	case resp.StatusCode == http.StatusOK:
		return body, -1, nil
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return nil, retryAfter(resp.Header), newAPIError(resp.StatusCode, body)
	default:
		return nil, -1, newAPIError(resp.StatusCode, body)
	}
}

func retryAfter(header http.Header) time.Duration {
	seconds, err := strconv.Atoi(header.Get("Retry-After"))
	if err != nil || seconds <= 0 {
		return 0
	}
	return min(time.Duration(seconds)*time.Second, maxRetryWait)
}

func (client *PinterestClient) FetchUserAccount(ctx context.Context) (*UserAccount, error) {
	bytes, err := client.ExecuteRequest(ctx, "/user_account")
	if err != nil {
		return nil, err
	}

	userAccount := new(UserAccount)
	if err := json.Unmarshal(bytes, userAccount); err != nil {
		return nil, err
	}

	return userAccount, nil
}

func (client *PinterestClient) FetchBoards(ctx context.Context) ([]Board, error) {
	return fetchAllPages[Board](ctx, client, "/boards/?page_size=100")
}

func (client *PinterestClient) FetchAllPins(ctx context.Context, board Board) ([]Pin, error) {
	return fetchAllPages[Pin](ctx, client, "/boards/"+url.PathEscape(board.Id)+"/pins?page_size=100")
}

// fetchAllPages follows bookmarks until the list is exhausted. The endpoint must
// already have a query string.
func fetchAllPages[T any](ctx context.Context, client *PinterestClient, endpoint string) ([]T, error) {
	var items []T
	bookmark := ""

	for {
		pageEndpoint := endpoint
		if bookmark != "" {
			pageEndpoint += "&bookmark=" + url.QueryEscape(bookmark)
		}

		bytes, err := client.ExecuteRequest(ctx, pageEndpoint)
		if err != nil {
			return nil, err
		}

		var current page[T]
		if err := json.Unmarshal(bytes, &current); err != nil {
			return nil, err
		}

		items = append(items, current.Items...)

		// A repeated bookmark would otherwise loop forever.
		if current.Bookmark == "" || current.Bookmark == bookmark {
			return items, nil
		}
		bookmark = current.Bookmark
	}
}
