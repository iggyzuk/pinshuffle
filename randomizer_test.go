package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func makePins(board string, count int) []Pin {
	pins := make([]Pin, count)
	for i := range pins {
		pins[i] = Pin{Id: fmt.Sprintf("%s-%d", board, i), BoardId: board}
	}
	return pins
}

func countByBoard(pins []Pin) map[string]int {
	counts := map[string]int{}
	for _, pin := range pins {
		counts[pin.BoardId]++
	}
	return counts
}

func TestSelectPinsReachesMaxWithUnevenBoards(t *testing.T) {
	boards := [][]Pin{makePins("a", 2), makePins("b", 100), makePins("c", 100)}

	selected := selectPins(boards, 100)

	if len(selected) != 100 {
		t.Fatalf("got %d pins", len(selected))
	}
	counts := countByBoard(selected)
	if counts["a"] != 2 || counts["b"] != 49 || counts["c"] != 49 {
		t.Fatalf("got %v", counts)
	}
}

func TestSelectPinsHonorsMaxBelowBoardCount(t *testing.T) {
	boards := [][]Pin{makePins("a", 5), makePins("b", 5), makePins("c", 5)}

	if got := len(selectPins(boards, 1)); got != 1 {
		t.Fatalf("got %d pins, a rounded down share would give 0", got)
	}
}

func TestSelectPinsHandlesEmptyInput(t *testing.T) {
	if got := selectPins([][]Pin{nil, {}}, 100); len(got) != 0 {
		t.Fatalf("got %d pins", len(got))
	}
	if got := selectPins(nil, 100); len(got) != 0 {
		t.Fatalf("got %d pins", len(got))
	}
	if got := selectPins([][]Pin{makePins("a", 3)}, 0); len(got) != 0 {
		t.Fatalf("got %d pins", len(got))
	}
}

func TestSelectPinsMixesBoards(t *testing.T) {
	boards := [][]Pin{makePins("a", 50), makePins("b", 50)}

	selected := selectPins(boards, 100)

	runLength := 1
	for i := 1; i < len(selected); i++ {
		if selected[i].BoardId == selected[i-1].BoardId {
			runLength++
		}
	}
	if runLength > 90 {
		t.Fatal("pins are still grouped by board")
	}
}

func boardsHandler(fail string, active *atomic.Int32, peak *atomic.Int32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := active.Add(1)
		defer active.Add(-1)
		for {
			seen := peak.Load()
			if now <= seen || peak.CompareAndSwap(seen, now) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)

		if strings.Contains(r.URL.Path, fail) {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"message":"nope"}`)
			return
		}
		fmt.Fprint(w, `{"items":[{"id":"1","board_id":"x"}]}`)
	}
}

func TestPartialFailureKeepsOtherBoards(t *testing.T) {
	var active, peak atomic.Int32
	client := newTestClient(t, boardsHandler("broken", &active, &peak))

	sources := []PinSource{{BoardId: "ok"}, {BoardId: "broken"}}
	pins, err := NewRandomizer(client, NewTaskStore().New("key")).GetRandomizedPins(context.Background(), 10, sources)

	if len(pins) != 1 {
		t.Fatalf("got %d pins", len(pins))
	}
	if err == nil || !strings.Contains(err.Error(), "1 of 2 selected") {
		t.Fatalf("got %v", err)
	}
}

func TestEveryFailureIsReportedWithoutLeaking(t *testing.T) {
	var active, peak atomic.Int32
	client := newTestClient(t, boardsHandler("boards", &active, &peak))

	sources := make([]PinSource, 20)
	for i := range sources {
		sources[i] = PinSource{BoardId: fmt.Sprint(i)}
	}

	_, err := NewRandomizer(client, NewTaskStore().New("key")).GetRandomizedPins(context.Background(), 10, sources)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("got %v", err)
	}
	if peak.Load() > maxConcurrentBoards {
		t.Fatalf("%d concurrent requests", peak.Load())
	}
}

func TestSectionPinsComeFromTheSection(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/boards/b1/sections/s1/pins" {
			t.Errorf("requested %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"items":[{"id":"1"}]}`)
	})

	pins, err := NewRandomizer(client, NewTaskStore().New("key")).GetRandomizedPins(context.Background(), 10, []PinSource{{BoardId: "b1", SectionId: "s1"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(pins) != 1 || pins[0].BoardId != "b1" || pins[0].BoardSectionId != "s1" {
		t.Fatalf("got %+v", pins)
	}
}
