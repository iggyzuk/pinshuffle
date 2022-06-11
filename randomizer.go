package main

import (
	"math/rand"
	"sync"
	"time"
)

type Randomizer struct {
	BoardIds     []string
	Max          int
	PinsPerBoard int
}

func NewRandomizer() *Randomizer {
	return &Randomizer{}
}

func (rnd *Randomizer) GetRandomizedPins(max int, boardIds []string) []Pin {
	rand.Seed(time.Now().UnixNano())
	rnd.Max = max
	rnd.ProccessBoards(boardIds)
	return rnd.FetchAllPinsFromSelectedBoards()
}

func (rnd *Randomizer) ProccessBoards(boardIds []string) {
	rnd.BoardIds = boardIds

	boardCount := len(rnd.BoardIds)

	if boardCount > 0 {
		rnd.PinsPerBoard = rnd.Max / boardCount
	}
}

func (rnd *Randomizer) FetchAllPinsFromSelectedBoards() []Pin {

	wg := sync.WaitGroup{}
	pins := []Pin{}

	for _, boardId := range rnd.BoardIds {
		id := boardId

		go func() {
			newPins := rnd.FetchPinsFromBoard(clientBoards[id])
			pins = append(pins, newPins...)
			wg.Done()
		}()

		wg.Add(1)
	}

	wg.Wait()

	return pins
}

func (rnd *Randomizer) FetchPinsFromBoard(board *Board) []Pin {
	allPins := client.FetchAllPins(board)
	trimmedPins := rnd.Trim(allPins, rnd.PinsPerBoard)
	return trimmedPins
}

func (rnd *Randomizer) Trim(pins []Pin, limit int) []Pin {
	for len(pins) > limit {
		pins = rnd.Remove(pins, rand.Intn(len(pins)))
	}
	return pins
}

func (rnd *Randomizer) Remove(pins []Pin, i int) []Pin {
	pins[i] = pins[len(pins)-1]
	// We do not need to put s[i] at the end, as it will be discarded anyway
	return pins[:len(pins)-1]
}
