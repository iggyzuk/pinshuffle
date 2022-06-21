package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

type Randomizer struct {
	Client       *PinterestClient
	ClientBoards map[string]*Board
	BoardIds     []string
	Max          int
	PinsPerBoard int
}

type PinsResult struct {
	Pins  []Pin
	Error error
}

func NewRandomizer(client *PinterestClient, clientBoards map[string]*Board) *Randomizer {
	return &Randomizer{Client: client, ClientBoards: clientBoards}
}

func (rnd *Randomizer) GetRandomizedPins(max int, boardIds []string) []Pin {
	rand.Seed(time.Now().UnixNano())
	rnd.Max = max
	rnd.ProccessBoards(boardIds)
	return rnd.FetchPinsFromSelectedBoards()
}

func (rnd *Randomizer) ProccessBoards(boardIds []string) {
	rnd.BoardIds = boardIds

	boardCount := len(rnd.BoardIds)

	if boardCount > 0 {
		rnd.PinsPerBoard = rnd.Max / boardCount
	}
}

func (rnd *Randomizer) FetchPinsFromSelectedBoards() []Pin {

	resultChan := make(chan PinsResult)

	for _, boardId := range rnd.BoardIds {

		go func(id string) {
			pins, err := rnd.FetchSomePinsFromBoard(rnd.ClientBoards[id])
			resultChan <- PinsResult{Pins: pins, Error: err}
		}(boardId)
	}

	allPins := []Pin{}

	for i := 0; i < len(rnd.BoardIds); i++ {

		r := <-resultChan

		if r.Error != nil {
			log.Fatal(r.Error)
			break
		}
		allPins = append(allPins, r.Pins...)
	}

	return allPins
}

func (rnd *Randomizer) FetchSomePinsFromBoard(board *Board) ([]Pin, error) {
	fmt.Println("Fetching all pins from Board: " + board.Name)
	allPins, err := rnd.Client.FetchAllPins(board)
	if err != nil {
		return nil, err
	}
	trimmedPins := rnd.Trim(allPins, rnd.PinsPerBoard)
	return trimmedPins, nil
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
