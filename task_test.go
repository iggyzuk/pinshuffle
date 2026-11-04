package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/template/html/v3"
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
	tc.AddPin(&Pin{Id: "1", BoardId: "gone", Media: Media{Images: Images{Res400x300: Image{Url: "u"}}}})

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

func TestImageFallsBackToAvailableSizes(t *testing.T) {
	onlySmall := Images{Res150x150: Image{Url: "small"}}
	noLargest := Images{Res150x150: Image{Url: "small"}, Res600x: Image{Url: "large"}}

	tests := []struct {
		resolution int
		images     Images
		want       string
	}{
		{1, onlySmall, "small"},
		{3, onlySmall, "small"},
		{3, noLargest, "large"},
		{1, noLargest, "large"},
		{2, Images{}, ""},
	}

	for _, test := range tests {
		query := TemplateUrlQuery{ImageResolution: test.resolution}
		if got := query.GetImageResolution(test.images).Url; got != test.want {
			t.Errorf("res %d: got %q, want %q", test.resolution, got, test.want)
		}
	}
}

func TestPopupGetsTheLargestImageWhateverTheGridUses(t *testing.T) {
	tc := NewTemplateController()
	tc.Model.UrlQuery.ImageResolution = 0
	tc.AddPin(&Pin{Media: Media{Images: Images{
		Res150x150: Image{Url: "small"},
		Res1200x:   Image{Url: "huge"},
	}}})

	if pin := tc.Model.Pins[0]; pin.ImageURL != "small" || pin.FullURL != "huge" {
		t.Fatalf("got %+v", pin)
	}
}

func TestPinWithoutAnyImageIsSkipped(t *testing.T) {
	tc := NewTemplateController()
	tc.AddPin(&Pin{Id: "1"})

	if len(tc.Model.Pins) != 0 {
		t.Fatal("would render a blank square")
	}
}

func TestParseQuerySections(t *testing.T) {
	boards := map[string]Board{"a": {Id: "a"}, "b": {Id: "b"}}

	query := parseQuery(t, "b=a&s=a:1&s=b:2&s=b:2&s=unknown:3&s=nonsense", boards)

	if len(query.Sections) != 1 || query.Sections[0] != (PinSource{BoardId: "b", SectionId: "2"}) {
		t.Fatalf("got %+v", query.Sections)
	}
}

func TestLoadSectionsDropsUnknownOnes(t *testing.T) {
	tc := NewTemplateController()
	tc.AddBoard(&TemplateBoard{Id: "b"})
	tc.AddBoard(&TemplateBoard{Id: "untouched"})
	tc.Model.UrlQuery.Sections = []PinSource{{BoardId: "b", SectionId: "1"}, {BoardId: "b", SectionId: "gone"}}

	var requested []string
	err := tc.LoadSections(context.Background(), func(_ context.Context, boardId string) ([]Section, error) {
		requested = append(requested, boardId)
		return []Section{{Id: "1", Name: "one"}, {Id: "2", Name: "two"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(requested) != 1 {
		t.Fatalf("requested %v, should only load boards that need it", requested)
	}
	if len(tc.Model.UrlQuery.Sections) != 1 || tc.Model.UrlQuery.Sections[0].SectionId != "1" {
		t.Fatalf("got %+v", tc.Model.UrlQuery.Sections)
	}
	if sections := tc.Model.Boards["b"].Sections; len(sections) != 2 || !sections[0].Selected || sections[1].Selected {
		t.Fatalf("got %+v", sections)
	}
}

func TestPinNamesItsSection(t *testing.T) {
	tc := NewTemplateController()
	tc.AddBoard(&TemplateBoard{Id: "b", Name: "Board", Sections: []TemplateSection{{Id: "s", Name: "Section"}}})
	tc.AddPin(&Pin{BoardId: "b", BoardSectionId: "s", Media: Media{Images: Images{Res400x300: Image{Url: "u"}}}})

	if got := tc.Model.Pins[0].Source(); got != "Board › Section" {
		t.Fatalf("got %q", got)
	}
}

func TestTaskKeyIgnoresBoardOrder(t *testing.T) {
	a := taskKey(TemplateUrlQuery{Max: 1, Boards: []string{"x", "y"}})
	b := taskKey(TemplateUrlQuery{Max: 1, Boards: []string{"y", "x"}})
	c := taskKey(TemplateUrlQuery{Max: 2, Boards: []string{"x", "y"}})

	if a != b || a == c {
		t.Fatalf("%q %q %q", a, b, c)
	}

	whole := taskKey(TemplateUrlQuery{Boards: []string{"x"}})
	section := taskKey(TemplateUrlQuery{Sections: []PinSource{{BoardId: "x", SectionId: "1"}}})
	if whole == section {
		t.Fatal("a section isn't the same as its board")
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

	task.Process(client, TemplateUrlQuery{Max: 1, Boards: []string{"a"}})

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
				items = append(items, fmt.Sprintf(`{"id":"%s-%d","board_id":%q,"media":{"images":{"400x300":{"url":"u"}}}}`, token, i, board))
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

func TestSectionShuffleFlow(t *testing.T) {
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/user_account":
			fmt.Fprint(w, `{"username":"alice"}`)
		case r.URL.Path == "/boards/b1/sections":
			fmt.Fprint(w, `{"items":[{"id":"s1","name":"Salads"},{"id":"s2","name":"Soups"}]}`)
		case r.URL.Path == "/boards/b1/sections/s1/pins":
			fmt.Fprint(w, `{"items":[{"id":"from-section","media":{"images":{"400x300":{"url":"u"}}}}]}`)
		case strings.HasSuffix(r.URL.Path, "/pins"):
			t.Errorf("fetched %s, only the section was selected", r.URL.Path)
		default:
			fmt.Fprint(w, `{"items":[{"id":"b1","name":"Food"}]}`)
		}
	}))

	session := map[string]string{accessTokenCookie: "alice"}
	target := "/?s=b1:s1"

	first := request(t, server, "GET", target, session)
	task := first.cookies[taskCookie]
	if task == nil {
		t.Fatalf("no task started: %s", first.body)
	}
	waitForTask(t, task.Value)

	session[taskCookie] = task.Value
	second := request(t, server, "GET", target, session)

	if !strings.Contains(second.body, `"Id":"from-section"`) || !strings.Contains(second.body, `"Section":"Salads"`) {
		t.Fatalf("section pin missing: %s", second.body)
	}
}

func TestSectionsEndpoint(t *testing.T) {
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/boards/full/sections":
			fmt.Fprint(w, `{"items":[{"id":"s1","name":"Salads"}]}`)
		case "/boards/empty/sections":
			fmt.Fprint(w, `{"items":[]}`)
		case "/boards/revoked/sections":
			w.WriteHeader(http.StatusUnauthorized)
		default:
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	server.Get("/boards/:id/sections", sectionsHandler)
	app.Sections = GetSections

	session := map[string]string{accessTokenCookie: "alice"}

	tests := []struct {
		target  string
		cookies map[string]string
		status  int
		body    string
	}{
		{"/boards/full/sections", session, 200, `[{"id":"s1","name":"Salads"}]`},
		{"/boards/empty/sections", session, 200, `[]`},
		{"/boards/revoked/sections", session, 401, ""},
		{"/boards/other/sections", session, 502, ""},
		{"/boards/full/sections", nil, 401, ""},
	}

	for _, test := range tests {
		res := request(t, server, "GET", test.target, test.cookies)
		if res.status != test.status || (test.body != "" && res.body != test.body) {
			t.Errorf("%s: got %d %q", test.target, res.status, res.body)
		}
	}
}

func TestProgress(t *testing.T) {
	newTask := func(boards, sections, expected int) *Task {
		query := TemplateUrlQuery{}
		for i := range boards {
			query.Boards = append(query.Boards, fmt.Sprint(i))
		}
		for i := range sections {
			query.Sections = append(query.Sections, PinSource{BoardId: "b", SectionId: fmt.Sprint(i)})
		}
		task := NewTaskStore().New("key")
		task.Describe(query, expected)
		return task
	}

	t.Run("follows the pins when their number is known", func(t *testing.T) {
		task := newTask(2, 0, 1000)
		task.AddPins(250)
		if got := task.Progress(); got.Percent != 25 || got.Unit != "boards" || got.Pins != 250 {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("follows the boards otherwise", func(t *testing.T) {
		task := newTask(2, 2, 0)
		task.AddPins(250)
		task.SourceDone()
		if got := task.Progress(); got.Percent != 25 || got.Unit != "boards and sections" {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("doesn't get stuck on a wrong count", func(t *testing.T) {
		task := newTask(2, 0, 1000)
		task.AddPins(10)
		task.SourceDone()
		if got := task.Progress(); got.Percent != 50 {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("only reaches 100 when complete", func(t *testing.T) {
		task := newTask(1, 0, 10)
		task.AddPins(50)
		task.SourceDone()
		if got := task.Progress(); got.Percent != 99 || got.Status != "processing" {
			t.Fatalf("got %+v", got)
		}
		close(task.done)
		if got := task.Progress(); got.Percent != 100 || got.Status != "complete" {
			t.Fatalf("got %+v", got)
		}
	})
}

func TestProgressDuringARealFetch(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("bookmark") == "" {
			fmt.Fprint(w, `{"items":[{"id":"1"},{"id":"2"}],"bookmark":"next"}`)
			return
		}
		fmt.Fprint(w, `{"items":[{"id":"3"}]}`)
	})

	query := TemplateUrlQuery{Max: 10, Boards: []string{"a", "b"}}
	task := NewTaskStore().New("key")
	task.Describe(query, 6)
	task.Process(client, query)

	if got := task.Progress(); got.Pins != 6 || got.Done != 2 || got.Percent != 100 {
		t.Fatalf("got %+v", got)
	}
}

func TestTaskEndpoint(t *testing.T) {
	server := newTestServer(t, fullPinterest(0))
	server.Get("/task/*", taskHandler)

	task := app.Tasks.New("key")
	task.Describe(TemplateUrlQuery{Boards: []string{"a", "b"}}, 100)
	task.AddPins(30)

	res := request(t, server, "GET", "/task/"+task.Id, nil)
	want := `{"status":"processing","percent":30,"done":0,"total":2,"unit":"boards","pins":30,"expected":100}`
	if res.body != want {
		t.Fatalf("got %s", res.body)
	}

	if res := request(t, server, "GET", "/task/unknown", nil); !strings.Contains(res.body, `"status":"error"`) {
		t.Fatalf("got %s", res.body)
	}
}

func TestBuildInfo(t *testing.T) {
	version, buildTime = "abc1234", "2026-11-04T15:30:00Z"
	defer func() { version, buildTime = "dev", "" }()

	info := currentBuild()

	if info.Time != "4 Nov 2026, 15:30 UTC" || info.URL != repoURL+"/commit/abc1234" {
		t.Fatalf("got %+v", info)
	}
}

func TestLocalBuildHasNoCommitLinkButStillADate(t *testing.T) {
	info := currentBuild()

	if info.Version != "dev" || info.URL != "" || info.Time == "" {
		t.Fatalf("got %+v", info)
	}
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
	tc.AddPin(&Pin{Id: "2", BoardId: "b", DominantColor: "#aabbcc", Media: Media{Images: Images{Res150x150: Image{Url: "https://i.pinimg.com/b.jpg"}}}})

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
