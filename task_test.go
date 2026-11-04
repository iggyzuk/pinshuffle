package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/template/html/v2"
	"github.com/valyala/fasthttp"
)

func parseQuery(t *testing.T, raw string, boards map[string]Board) TemplateUrlQuery {
	t.Helper()
	uri := fasthttp.AcquireURI()
	defer fasthttp.ReleaseURI(uri)
	uri.SetQueryString(raw)

	tc := NewTemplateController()
	if err := tc.ParseUrlQueries(uri, boards); err != nil {
		t.Fatal(err)
	}
	return tc.Model.UrlQuery
}

func TestParseQueryLimits(t *testing.T) {
	tests := []struct {
		raw     string
		wantMax int
		wantRes int
	}{
		{"", 100, 1},
		{"max=50&res=3", 50, 3},
		{"max=-1", 1, 1},
		{"max=0", 1, 1},
		{"max=99999999", 10000, 1},
		{"max=lots&res=big", 100, 1},
		{"res=-5", 100, 0},
		{"res=9", 100, 3},
	}

	for _, test := range tests {
		query := parseQuery(t, test.raw, nil)
		if query.Max != test.wantMax || query.ImageResolution != test.wantRes {
			t.Errorf("%q: got max=%d res=%d", test.raw, query.Max, query.ImageResolution)
		}
	}
}

func TestParseQueryBoards(t *testing.T) {
	boards := map[string]Board{"a": {Id: "a"}, "b": {Id: "b"}}

	query := parseQuery(t, "b=a&b=a&b=unknown&b=b", boards)

	if strings.Join(query.Boards, ",") != "a,b" {
		t.Fatalf("got %v", query.Boards)
	}
}

func TestPinOfDeletedBoardStillRenders(t *testing.T) {
	tc := NewTemplateController()
	tc.AddPin(&Pin{Id: "1", BoardId: "gone"})

	if tc.Model.Pins[0].Board == nil {
		t.Fatal("template would fail on a nil board")
	}
}

func TestPinCarriesTheShapeOfTheChosenImage(t *testing.T) {
	tc := NewTemplateController()
	tc.Model.UrlQuery.ImageResolution = 2
	tc.AddPin(&Pin{Media: Media{Images: Images{
		Res400x300: Image{Width: 400, Height: 300},
		Res600x:    Image{Url: "u", Width: 600, Height: 900},
	}}})

	pin := tc.Model.Pins[0]
	if pin.ImageURL != "u" || pin.Width != 600 || pin.Height != 900 {
		t.Fatalf("got %+v", pin)
	}
}

func TestTaskKeyIgnoresBoardOrder(t *testing.T) {
	a := taskKey(TemplateUrlQuery{Max: 1, Boards: []string{"x", "y"}})
	b := taskKey(TemplateUrlQuery{Max: 1, Boards: []string{"y", "x"}})
	c := taskKey(TemplateUrlQuery{Max: 2, Boards: []string{"x", "y"}})

	if a != b || a == c {
		t.Fatalf("%q %q %q", a, b, c)
	}
}

func TestTaskStoreConcurrentUse(t *testing.T) {
	store := NewTaskStore()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			task := store.New("key")
			store.Get(task.Id)
			store.DeleteExpired(time.Now())
			store.Delete(task.Id)
		}()
	}

	wg.Wait()
}

func TestTaskStoreDropsExpiredTasks(t *testing.T) {
	store := NewTaskStore()
	task := store.New("key")

	store.DeleteExpired(task.Timestamp.Add(taskTTL - time.Second))
	if _, exists := store.Get(task.Id); !exists {
		t.Fatal("deleted too early")
	}

	store.DeleteExpired(task.Timestamp.Add(taskTTL))
	if _, exists := store.Get(task.Id); exists {
		t.Fatal("not deleted")
	}
}

func TestTaskSurvivesPanicWhileFetching(t *testing.T) {
	task := NewTaskStore().New("key")

	// Blows up in the goroutine that fetches the board.
	client := NewClient()
	client.HttpClient = nil

	task.Process(client, map[string]Board{"a": {Id: "a"}}, TemplateUrlQuery{Max: 1, Boards: []string{"a"}})

	if !task.IsComplete() || !strings.Contains(task.Error, "panic") {
		t.Fatalf("complete: %t, error: %q", task.IsComplete(), task.Error)
	}
}

// fullPinterest serves one user's boards and pins, per token.
func fullPinterest(pinsPerBoard int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		switch {
		case r.URL.Path == "/user_account":
			fmt.Fprintf(w, `{"username":%q}`, token)
		case strings.HasSuffix(r.URL.Path, "/pins"):
			board := strings.Split(r.URL.Path, "/")[2]
			var items []string
			for i := 0; i < pinsPerBoard; i++ {
				items = append(items, fmt.Sprintf(`{"id":"%s-%d","board_id":%q}`, token, i, board))
			}
			fmt.Fprintf(w, `{"items":[%s]}`, strings.Join(items, ","))
		default:
			fmt.Fprintf(w, `{"items":[{"id":"%s-board","name":"board"}]}`, token)
		}
	}
}

func waitForTask(t *testing.T, id string) {
	t.Helper()
	task, exists := app.Tasks.Get(id)
	if !exists {
		t.Fatal("task not found")
	}
	select {
	case <-task.done:
	case <-time.After(5 * time.Second):
		t.Fatal("task timed out")
	}
}

func TestShuffleFlow(t *testing.T) {
	server := newTestServer(t, fullPinterest(3))

	session := map[string]string{accessTokenCookie: "alice"}
	target := "/?b=alice-board&max=2"

	first := request(t, server, "GET", target, session)
	task := first.cookies[taskCookie]
	if task == nil || !task.HttpOnly || !strings.Contains(first.body, `"Loading":true`) {
		t.Fatalf("no task started: %+v %s", task, first.body)
	}

	waitForTask(t, task.Value)

	session[taskCookie] = task.Value
	second := request(t, server, "GET", target, session)
	if !strings.Contains(second.body, `"Id":"alice-`) || strings.Count(second.body, `"ImageURL"`) != 2 {
		t.Fatalf("pins missing: %s", second.body)
	}
	if cookie := second.cookies[taskCookie]; cookie == nil || cookie.Value != "" {
		t.Fatalf("task cookie not cleared: %+v", cookie)
	}
	if _, exists := app.Tasks.Get(task.Value); exists {
		t.Fatal("task not consumed")
	}
}

func TestStaleTaskStartsOver(t *testing.T) {
	server := newTestServer(t, fullPinterest(1))

	res := request(t, server, "GET", "/?b=alice-board", map[string]string{accessTokenCookie: "alice", taskCookie: "lost-in-a-restart"})

	task := res.cookies[taskCookie]
	if task == nil || task.Value == "" || task.Value == "lost-in-a-restart" {
		t.Fatalf("got %+v", task)
	}
	waitForTask(t, task.Value)
}

func TestChangingSelectionStartsOver(t *testing.T) {
	server := newTestServer(t, fullPinterest(1))

	running := app.Tasks.New("another selection")
	close(running.done)

	res := request(t, server, "GET", "/?b=alice-board", map[string]string{accessTokenCookie: "alice", taskCookie: running.Id})

	task := res.cookies[taskCookie]
	if task == nil || task.Value == running.Id || task.Value == "" {
		t.Fatalf("got %+v", task)
	}
	waitForTask(t, task.Value)
}

func TestNegativeMaxDoesNotCrash(t *testing.T) {
	server := newTestServer(t, fullPinterest(0))

	res := request(t, server, "GET", "/?b=alice-board&max=-1", map[string]string{accessTokenCookie: "alice"})
	waitForTask(t, res.cookies[taskCookie].Value)
}

// Requests for different users overlapping used to swap their tokens, because
// the token lived on the one shared client.
func TestConcurrentUsersKeepTheirOwnAccounts(t *testing.T) {
	server := newTestServer(t, fullPinterest(1))
	var wg sync.WaitGroup

	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(user string) {
			defer wg.Done()

			res := request(t, server, "GET", "/", map[string]string{accessTokenCookie: url.QueryEscape(user)})

			if !strings.Contains(res.body, fmt.Sprintf(`"Name":%q`, user)) {
				t.Errorf("%s got someone else's page: %s", user, res.body)
			}
		}(fmt.Sprintf("user%d", i))
	}

	wg.Wait()
}

func TestTemplateRendersRealPins(t *testing.T) {
	engine := html.New("./templates", ".gohtml")
	engine.AddFunc("IsBoardSelected", IsBoardSelected)
	engine.AddFunc("SortBoards", SortBoards)

	tc := NewTemplateController()
	tc.Model.Authenticated = true
	tc.Model.TaskId = "the-task"
	tc.AddBoard(&TemplateBoard{Id: "b", Name: "Board"})
	tc.Model.UrlQuery.ImageResolution = 2
	tc.AddPin(&Pin{Id: "1", BoardId: "b", DominantColor: "#aabbcc", Media: Media{Images: Images{Res600x: Image{Url: "https://i.pinimg.com/a.jpg", Width: 600, Height: 900}}}})
	tc.AddPin(&Pin{Id: "2", BoardId: "b", DominantColor: "#aabbcc"})

	var out bytes.Buffer
	if err := engine.Render(&out, "layout", tc.Model); err != nil {
		t.Fatal(err)
	}
	page := out.String()

	for _, want := range []string{`data-task-id="the-task"`, `--pin-aspect: 600 / 900;`, `background-color:#aabbcc`, `data-pin-url="https://www.pinterest.com/pin/1"`} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %s", want)
		}
	}
	if strings.Contains(page, "ZgotmplZ") {
		t.Error("the template escaper rejected a value")
	}
	if got := strings.Count(page, "--pin-aspect"); got != 1 {
		t.Errorf("%d aspect ratios, the pin without a size shouldn't get one", got)
	}
}
