package main

import (
	"math/rand"
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

func (rnd *Randomizer) GetRandomizedPins(tm *TemplateModel) []Pin {
	rand.Seed(time.Now().UnixNano())
	rnd.Max = tm.UrlQuery.Max
	rnd.ProccessBoards(tm.UrlQuery)
	return rnd.FetchAllPinsFromSelectedBoards()
}

func (rnd *Randomizer) ProccessBoards(urlQueryModel *TemplateUrlQuery) {
	for _, queryBoard := range urlQueryModel.Boards {
		rnd.BoardIds = append(rnd.BoardIds, queryBoard)
	}
	boardCount := len(rnd.BoardIds)
	if boardCount > 0 {
		rnd.PinsPerBoard = rnd.Max / boardCount
	}
}

func (rnd *Randomizer) FetchAllPinsFromSelectedBoards() []Pin {

	ch := make(chan []Pin)

	for _, tb := range tm.Boards {
		go rnd.FetchPinsFromBoard(tb.Board, ch)
	}

	pins := []Pin{}
	newPins := <-ch
	pins = append(pins, newPins...)

	return pins
}

func (rnd *Randomizer) FetchPinsFromBoard(board *Board, ch chan []Pin) {
	allPins := client.FetchAllPins(board)
	trimmedPins := rnd.Trim(allPins, rnd.PinsPerBoard)
	ch <- trimmedPins
}

func (rnd *Randomizer) Trim(pins []Pin, limit int) []Pin {
	for len(pins) > limit {
		return rnd.Remove(pins, rand.Intn(len(pins)))
	}
	return pins
}

func (rnd *Randomizer) Remove(s []Pin, i int) []Pin {
	s[i] = s[len(s)-1]
	// We do not need to put s[i] at the end, as it will be discarded anyway
	return s[:len(s)-1]
}
