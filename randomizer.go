package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
)

// Boards are fetched in parallel, but a user can select hundreds of them and
// each needs several requests against Pinterest's rate limit.
const maxConcurrentBoards = 4

// PinSource is somewhere to get pins from: a whole board, or one of its sections
// if SectionId is set.
type PinSource struct {
	BoardId   string
	SectionId string
}

func (source PinSource) String() string {
	if source.SectionId != "" {
		return "section " + source.SectionId
	}
	return "board " + source.BoardId
}

func (source PinSource) fetch(ctx context.Context, client *PinterestClient, onPage func(count int)) ([]Pin, error) {
	if source.SectionId != "" {
		return client.FetchSectionPins(ctx, source.BoardId, source.SectionId, onPage)
	}
	return client.FetchAllPins(ctx, Board{Id: source.BoardId}, onPage)
}

// ProgressReporter hears how fetching is going.
type ProgressReporter interface {
	AddPins(count int)
	SourceDone()
}

type Randomizer struct {
	Client   *PinterestClient
	Progress ProgressReporter
}

func NewRandomizer(client *PinterestClient, progress ProgressReporter) *Randomizer {
	return &Randomizer{Client: client, Progress: progress}
}

// GetRandomizedPins fetches every pin of the sources and returns up to max of
// them in random order. If only some sources fail it returns the pins of the
// others along with the error.
func (rnd *Randomizer) GetRandomizedPins(ctx context.Context, max int, sources []PinSource) ([]Pin, error) {
	perSource := make([][]Pin, len(sources))
	errs := make([]error, len(sources))

	slots := make(chan struct{}, maxConcurrentBoards)
	var wg sync.WaitGroup

	for i, source := range sources {
		wg.Add(1)
		go func(i int, source PinSource) {
			defer wg.Done()
			defer rnd.Progress.SourceDone()

			// A panic here is out of reach of the task's recover, and would take
			// down the server for everyone.
			defer func() {
				if r := recover(); r != nil {
					errs[i] = fmt.Errorf("panic while loading %s: %v", source, r)
				}
			}()

			slots <- struct{}{}
			defer func() { <-slots }()

			perSource[i], errs[i] = source.fetch(ctx, rnd.Client, rnd.Progress.AddPins)
			log.Printf("Received %d pins from %s", len(perSource[i]), source)
		}(i, source)
	}

	wg.Wait()

	return selectPins(perSource, max), summarizeErrors(errs)
}

// selectPins takes pins from each source in turn so that small ones aren't
// drowned out, and sources with spare pins fill the share of those that ran out.
func selectPins(perSource [][]Pin, max int) []Pin {
	for _, pins := range perSource {
		shufflePins(pins)
	}
	rand.Shuffle(len(perSource), func(i, j int) {
		perSource[i], perSource[j] = perSource[j], perSource[i]
	})

	selected := []Pin{}

	for depth := 0; len(selected) < max; depth++ {
		exhausted := true

		for _, pins := range perSource {
			if depth < len(pins) && len(selected) < max {
				selected = append(selected, pins[depth])
				exhausted = false
			}
		}

		if exhausted {
			break
		}
	}

	shufflePins(selected)

	return selected
}

func shufflePins(pins []Pin) {
	rand.Shuffle(len(pins), func(i, j int) {
		pins[i], pins[j] = pins[j], pins[i]
	})
}

func summarizeErrors(errs []error) error {
	var first error
	failed := 0

	for _, err := range errs {
		if err != nil {
			failed++
			if first == nil {
				first = err
			}
		}
	}

	if first == nil {
		return nil
	}
	return fmt.Errorf("%d of %d selected boards and sections failed to load: %w", failed, len(errs), first)
}
