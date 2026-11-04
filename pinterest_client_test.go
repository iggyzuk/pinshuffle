package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *PinterestClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := NewClient()
	client.BaseURL = server.URL
	client.RetryDelay = time.Millisecond
	return client
}

func TestBookmarkIsEscaped(t *testing.T) {
	const bookmark = "a+b/c=="
	var received []string

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		received = append(received, r.URL.Query().Get("bookmark"))
		if len(received) == 1 {
			fmt.Fprintf(w, `{"items":[{"id":"1"}],"bookmark":%q}`, bookmark)
			return
		}
		fmt.Fprint(w, `{"items":[{"id":"2"}],"bookmark":null}`)
	})

	boards, err := client.FetchBoards(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(boards) != 2 {
		t.Fatalf("got %d boards, want 2", len(boards))
	}
	if received[1] != bookmark {
		t.Fatalf("server received bookmark %q, want %q", received[1], bookmark)
	}
}

func TestRepeatedBookmarkStopsPaging(t *testing.T) {
	var requests atomic.Int32

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, `{"items":[{"id":"1"}],"bookmark":"same"}`)
	})

	if _, err := client.FetchBoards(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("made %d requests, want 2", got)
	}
}

func TestRetriesRateLimit(t *testing.T) {
	var requests atomic.Int32

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		fmt.Fprint(w, `{"username":"iggy"}`)
	})

	user, err := client.FetchUserAccount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "iggy" {
		t.Fatalf("got user %q", user.Username)
	}
}

func TestDoesNotRetryUnauthorized(t *testing.T) {
	var requests atomic.Int32

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"code":2,"message":"Authentication failed."}`)
	})

	_, err := client.FetchUserAccount(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("got %v, want ErrUnauthorized", err)
	}
	if !strings.Contains(err.Error(), "Authentication failed.") {
		t.Fatalf("error %q lacks Pinterest's message", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("made %d requests, want 1", got)
	}
}

func TestWithTokenDoesNotShareTheToken(t *testing.T) {
	client := NewClient()
	alice := client.WithToken("alice")
	bob := client.WithToken("bob")

	if alice.AccessToken != "alice" || bob.AccessToken != "bob" || client.AccessToken != "" {
		t.Fatalf("tokens leaked: %q %q %q", alice.AccessToken, bob.AccessToken, client.AccessToken)
	}
}

func TestAuthUriEscapesRedirectAndCarriesState(t *testing.T) {
	uri := NewClient().GetAuthUri("my-state")

	for _, want := range []string{"state=my-state", "redirect_uri=https%3A%2F%2Fpinshuffle.fly.dev%2Fredirect%2F"} {
		if !strings.Contains(uri, want) {
			t.Errorf("%s lacks %s", uri, want)
		}
	}
}

func TestCodeExchange(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if got := r.Form.Get("redirect_uri"); got != "https://pinshuffle.fly.dev/redirect/" {
			t.Errorf("redirect_uri = %q, must match the authorize step", got)
		}
		if got := r.Form.Get("continuous_refresh"); got != "true" {
			t.Errorf("continuous_refresh = %q", got)
		}
		fmt.Fprint(w, `{"access_token":"pina","refresh_token":"pinr","expires_in":2592000,"refresh_token_expires_in":5184000}`)
	})

	tokens, err := client.FetchAccessToken(context.Background(), "code")
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "pina" || tokens.RefreshToken != "pinr" {
		t.Fatalf("got %+v", tokens)
	}
	if left := time.Until(tokens.ExpiresAt); left < 29*24*time.Hour || left > 31*24*time.Hour {
		t.Fatalf("access token expires in %s", left)
	}
	if left := time.Until(tokens.RefreshExpiresAt); left < 59*24*time.Hour || left > 61*24*time.Hour {
		t.Fatalf("refresh token expires in %s", left)
	}
}

func TestRefreshRotatesTokens(t *testing.T) {
	expiresAt := time.Now().Add(60 * 24 * time.Hour).Unix()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "old-refresh" {
			t.Errorf("bad form: %v", r.Form)
		}
		if id, secret, ok := r.BasicAuth(); !ok || id != "id" || secret != "secret" {
			t.Errorf("bad basic auth: %q %q", id, secret)
		}
		fmt.Fprintf(w, `{"access_token":"new","refresh_token":"new-refresh","expires_in":2592000,"refresh_token_expires_at":%d}`, expiresAt)
	})
	client.AppID, client.Secret = "id", "secret"

	tokens, err := client.RefreshTokens(context.Background(), Tokens{RefreshToken: "old-refresh"})
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "new" || tokens.RefreshToken != "new-refresh" {
		t.Fatalf("got %+v", tokens)
	}
	if tokens.RefreshExpiresAt.Unix() != expiresAt {
		t.Fatalf("refresh expiry %v", tokens.RefreshExpiresAt)
	}
}

func TestRefreshKeepsTheOldRefreshTokenIfNoneIsReturned(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"access_token":"new","expires_in":2592000}`)
	})

	tokens, err := client.RefreshTokens(context.Background(), Tokens{RefreshToken: "old-refresh"})
	if err != nil {
		t.Fatal(err)
	}
	if tokens.RefreshToken != "old-refresh" {
		t.Fatalf("got %+v", tokens)
	}
}

func TestRefreshErrors(t *testing.T) {
	tests := []struct {
		status       int
		unauthorized bool
	}{
		{http.StatusBadRequest, true},
		{http.StatusUnauthorized, true},
		{http.StatusTooManyRequests, false},
		{http.StatusInternalServerError, false},
	}

	for _, test := range tests {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(test.status)
		})

		_, err := client.RefreshTokens(context.Background(), Tokens{RefreshToken: "r"})
		if err == nil || errors.Is(err, ErrUnauthorized) != test.unauthorized {
			t.Errorf("status %d: got %v, unauthorized should be %t", test.status, err, test.unauthorized)
		}
	}
}

func TestConcurrentRefreshesSpendTheTokenOnce(t *testing.T) {
	var requests atomic.Int32
	release := make(chan struct{})

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		<-release
		fmt.Fprint(w, `{"access_token":"new","expires_in":2592000}`)
	})

	results := make(chan error, 5)
	for i := 0; i < 5; i++ {
		go func() {
			_, err := client.WithToken("").RefreshTokens(context.Background(), Tokens{RefreshToken: "r"})
			results <- err
		}()
	}

	time.Sleep(50 * time.Millisecond)
	close(release)

	for i := 0; i < 5; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("made %d refresh requests, want 1", got)
	}
}
