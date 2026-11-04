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

type Randomizer struct {
	Client       *PinterestClient
	ClientBoards map[string]Board
}

func NewRandomizer(client *PinterestClient, clientBoards map[string]Board) *Randomizer {
	return &Randomizer{Client: client, ClientBoards: clientBoards}
}

// GetRandomizedPins fetches every pin of the selected boards and returns up to
// max of them in random order. If only some boards fail it returns the pins of
// the others along with the error.
func (rnd *Randomizer) GetRandomizedPins(ctx context.Context, max int, boardIds []string) ([]Pin, error) {
	perBoard := make([][]Pin, len(boardIds))
	errs := make([]error, len(boardIds))

	slots := make(chan struct{}, maxConcurrentBoards)
	var wg sync.WaitGroup

	for i, boardId := range boardIds {
		wg.Add(1)
		go func(i int, board Board) {
			defer wg.Done()

			// A panic here is out of reach of the task's recover, and would take
			// down the server for everyone.
			defer func() {
				if r := recover(); r != nil {
					errs[i] = fmt.Errorf("panic while loading board %s: %v", board.Id, r)
				}
			}()

			slots <- struct{}{}
			defer func() { <-slots }()

			perBoard[i], errs[i] = rnd.Client.FetchAllPins(ctx, board)
			log.Printf("Received %d pins from board %s", len(perBoard[i]), board.Id)
		}(i, rnd.ClientBoards[boardId])
	}

	wg.Wait()

	return selectPins(perBoard, max), summarizeErrors(errs)
}

// selectPins takes pins from each board in turn so that small boards aren't
// drowned out, and boards with spare pins fill the share of those that ran out.
func selectPins(perBoard [][]Pin, max int) []Pin {
	for _, pins := range perBoard {
		shufflePins(pins)
	}
	rand.Shuffle(len(perBoard), func(i, j int) {
		perBoard[i], perBoard[j] = perBoard[j], perBoard[i]
	})

	selected := []Pin{}

	for depth := 0; len(selected) < max; depth++ {
		exhausted := true

		for _, pins := range perBoard {
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
	return fmt.Errorf("%d of %d boards failed to load: %w", failed, len(errs), first)
}
