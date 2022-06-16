package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Randomizer struct {
	Client       *PinterestClient
	ClientBoards map[string]*Board
	BoardIds     []string
	Max          int
	PinsPerBoard int
}

func NewRandomizer(client *PinterestClient, clientBoards map[string]*Board) *Randomizer {
	return &Randomizer{Client: client, ClientBoards: clientBoards}
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

		go func(id string) {
			newPins := rnd.FetchPinsFromBoard(rnd.ClientBoards[id])
			pins = append(pins, newPins...)
			wg.Done()
		}(boardId)

		wg.Add(1)
	}

	wg.Wait()

	return pins
}

func (rnd *Randomizer) FetchPinsFromBoard(board *Board) []Pin {
	fmt.Println("Fetching all pins from Board: " + board.Name)
	allPins, err := rnd.Client.FetchAllPins(board)
	if err != nil {
		fmt.Println(err.Error())
	}
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
