package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestNeedsRefresh(t *testing.T) {
	now := time.Now()
	day := 24 * time.Hour

	tests := []struct {
		name   string
		tokens Tokens
		want   bool
	}{
		{"plenty of time", Tokens{AccessToken: "a", RefreshToken: "r", ExpiresAt: now.Add(20 * day)}, false},
		{"close to expiry", Tokens{AccessToken: "a", RefreshToken: "r", ExpiresAt: now.Add(3 * day)}, true},
		{"expired", Tokens{AccessToken: "a", RefreshToken: "r", ExpiresAt: now.Add(-day)}, true},
		{"access cookie gone", Tokens{RefreshToken: "r"}, true},
		{"cannot refresh", Tokens{AccessToken: "a", ExpiresAt: now.Add(-day)}, false},
		{"session from before refreshing", Tokens{AccessToken: "a"}, false},
	}

	for _, test := range tests {
		if got := test.tokens.NeedsRefresh(now); got != test.want {
			t.Errorf("%s: got %t", test.name, got)
		}
	}
}

// fakePinterest accepts valid tokens only, and counts refreshes.
type fakePinterest struct {
	valid     string
	refreshes atomic.Int32
	badGrant  bool
}

func (f *fakePinterest) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/oauth/token":
		f.refreshes.Add(1)
		if f.badGrant {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f.valid = "fresh"
		fmt.Fprint(w, `{"access_token":"fresh","refresh_token":"fresh-refresh","expires_in":2592000,"refresh_token_expires_in":5184000}`)
	case r.Header.Get("Authorization") != "Bearer "+f.valid:
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"code":2,"message":"Authentication failed."}`)
	case r.URL.Path == "/user_account":
		fmt.Fprint(w, `{"username":"iggy"}`)
	default:
		fmt.Fprint(w, `{"items":[{"id":"1","name":"board"}]}`)
	}
}

func newTestServer(t *testing.T, pinterest http.Handler) *fiber.App {
	t.Helper()

	fake := httptest.NewServer(pinterest)
	t.Cleanup(fake.Close)

	app = App{PinClient: NewClient(), Func: GetTemplateModel, Tasks: NewTaskStore()}
	app.PinClient.BaseURL = fake.URL
	app.PinClient.RetryDelay = time.Millisecond

	server := fiber.New(fiber.Config{Immutable: true})
	server.Get("/", func(c *fiber.Ctx) error {
		model, err := app.Func(c)
		if err != nil {
			return err
		}
		return c.JSON(model)
	})
	server.Get("/redirect", authRedirectHandler)
	server.Get("/login", loginHandler)
	server.Post("/logout", logoutHandler)
	return server
}

type result struct {
	status  int
	body    string
	cookies map[string]*http.Cookie
}

func request(t *testing.T, server *fiber.App, method, target string, cookies map[string]string) result {
	t.Helper()

	req := httptest.NewRequest(method, target, nil)
	for name, value := range cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}

	resp, err := server.Test(req, 5000)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	res := result{status: resp.StatusCode, body: string(body), cookies: map[string]*http.Cookie{}}
	for _, cookie := range resp.Cookies() {
		res.cookies[cookie.Name] = cookie
	}
	return res
}

func unix(t time.Time) string {
	return fmt.Sprint(t.Unix())
}

func TestExpiringTokenIsRefreshedBeforeUse(t *testing.T) {
	pinterest := &fakePinterest{valid: "stale"}
	server := newTestServer(t, pinterest)

	res := request(t, server, "GET", "/", map[string]string{
		accessTokenCookie:  "stale",
		refreshTokenCookie: "stale-refresh",
		tokenExpiryCookie:  unix(time.Now().Add(24 * time.Hour)),
	})

	if !strings.Contains(res.body, `"Name":"iggy"`) {
		t.Fatalf("not logged in: %s", res.body)
	}
	if pinterest.refreshes.Load() != 1 {
		t.Fatalf("%d refreshes", pinterest.refreshes.Load())
	}

	access := res.cookies[accessTokenCookie]
	if access == nil || access.Value != "fresh" || !access.HttpOnly || !access.Secure {
		t.Fatalf("access cookie: %+v", access)
	}
	if refresh := res.cookies[refreshTokenCookie]; refresh == nil || refresh.Value != "fresh-refresh" || !refresh.HttpOnly {
		t.Fatalf("refresh cookie: %+v", refresh)
	}
}

func TestValidTokenIsNotRefreshed(t *testing.T) {
	pinterest := &fakePinterest{valid: "good"}
	server := newTestServer(t, pinterest)

	res := request(t, server, "GET", "/", map[string]string{
		accessTokenCookie:  "good",
		refreshTokenCookie: "r",
		tokenExpiryCookie:  unix(time.Now().Add(20 * 24 * time.Hour)),
	})

	if pinterest.refreshes.Load() != 0 || res.cookies[accessTokenCookie] != nil {
		t.Fatalf("refreshed anyway: %d", pinterest.refreshes.Load())
	}
}

func TestExpiredAccessCookieIsRestoredFromRefreshToken(t *testing.T) {
	pinterest := &fakePinterest{}
	server := newTestServer(t, pinterest)

	res := request(t, server, "GET", "/", map[string]string{refreshTokenCookie: "r"})

	if !strings.Contains(res.body, `"Name":"iggy"`) {
		t.Fatalf("not logged in: %s", res.body)
	}
}

func TestRevokedTokenIsRefreshedAndRetried(t *testing.T) {
	pinterest := &fakePinterest{valid: "other"}
	server := newTestServer(t, pinterest)

	res := request(t, server, "GET", "/", map[string]string{
		accessTokenCookie:  "revoked",
		refreshTokenCookie: "r",
		tokenExpiryCookie:  unix(time.Now().Add(20 * 24 * time.Hour)),
	})

	if !strings.Contains(res.body, `"Name":"iggy"`) || pinterest.refreshes.Load() != 1 {
		t.Fatalf("%d refreshes: %s", pinterest.refreshes.Load(), res.body)
	}
}

func TestDeadSessionLogsOut(t *testing.T) {
	tests := map[string]map[string]string{
		"refresh rejected": {accessTokenCookie: "expired", refreshTokenCookie: "dead", tokenExpiryCookie: unix(time.Now().Add(-time.Hour))},
		"no refresh token": {accessTokenCookie: "expired"},
	}

	for name, cookies := range tests {
		t.Run(name, func(t *testing.T) {
			server := newTestServer(t, &fakePinterest{valid: "other", badGrant: true})

			res := request(t, server, "GET", "/", cookies)

			if !strings.Contains(res.body, `"Authenticated":false`) || !strings.Contains(res.body, "expired") {
				t.Fatalf("still logged in: %s", res.body)
			}
			if cookie := res.cookies[accessTokenCookie]; cookie == nil || cookie.Value != "" {
				t.Fatalf("access cookie not cleared: %+v", cookie)
			}
		})
	}
}

func TestFailedEarlyRefreshKeepsWorkingToken(t *testing.T) {
	pinterest := &fakePinterest{valid: "good", badGrant: true}
	server := newTestServer(t, pinterest)

	res := request(t, server, "GET", "/", map[string]string{
		accessTokenCookie:  "good",
		refreshTokenCookie: "r",
		tokenExpiryCookie:  unix(time.Now().Add(24 * time.Hour)),
	})

	if !strings.Contains(res.body, `"Name":"iggy"`) || res.cookies[accessTokenCookie] != nil {
		t.Fatalf("session lost: %s", res.body)
	}
}

func TestOAuthCallback(t *testing.T) {
	pinterest := &fakePinterest{}

	t.Run("state mismatch is rejected", func(t *testing.T) {
		server := newTestServer(t, pinterest)
		res := request(t, server, "GET", "/redirect?code=c&state=attacker", map[string]string{oauthStateCookie: "mine"})
		if res.cookies[accessTokenCookie] != nil || pinterest.refreshes.Load() != 0 {
			t.Fatal("logged in with a forged state")
		}
	})

	t.Run("missing state is rejected", func(t *testing.T) {
		server := newTestServer(t, pinterest)
		res := request(t, server, "GET", "/redirect?code=c", nil)
		if res.cookies[accessTokenCookie] != nil {
			t.Fatal("logged in without a state")
		}
	})

	t.Run("declining shows an error", func(t *testing.T) {
		server := newTestServer(t, pinterest)
		res := request(t, server, "GET", "/?auth=failed", nil)
		if !strings.Contains(res.body, "Could not sign in") {
			t.Fatal(res.body)
		}
	})

	t.Run("login round trip", func(t *testing.T) {
		server := newTestServer(t, pinterest)

		login := request(t, server, "GET", "/login", nil)
		state := login.cookies[oauthStateCookie]
		if state == nil || !state.HttpOnly {
			t.Fatalf("state cookie: %+v", state)
		}

		res := request(t, server, "GET", "/redirect/?code=c&state="+state.Value, map[string]string{oauthStateCookie: state.Value})
		if cookie := res.cookies[accessTokenCookie]; cookie == nil || cookie.Value != "fresh" {
			t.Fatalf("access cookie: %+v", cookie)
		}
	})
}

func TestLogoutClearsEverything(t *testing.T) {
	server := newTestServer(t, &fakePinterest{})

	res := request(t, server, "POST", "/logout", nil)

	for _, name := range []string{accessTokenCookie, refreshTokenCookie, tokenExpiryCookie, taskCookie} {
		cookie := res.cookies[name]
		if cookie == nil || cookie.Value != "" || cookie.Path != "/" || cookie.Expires.After(time.Now()) {
			t.Errorf("%s not cleared: %+v", name, cookie)
		}
	}
}

func TestMain(m *testing.M) {
	os.Unsetenv("BASE_URL")
	os.Exit(m.Run())
}
