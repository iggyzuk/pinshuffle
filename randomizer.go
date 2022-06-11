package main

import (
	"math/rand"
	"time"
)

type Randomizer struct {
	Boards       []string
	Max          int
	PinsPerBoard int
}

func NewRandomizer(urlQueryModel *TemplateUrlQuery) *Randomizer {
	rnd := &Randomizer{Max: urlQueryModel.Max}
	rnd.ProccessBoards(urlQueryModel)
	return rnd
}

func (r *Randomizer) ProccessBoards(urlQueryModel *TemplateUrlQuery) {
	for _, queryBoard := range urlQueryModel.Boards {
		r.Boards = append(r.Boards, queryBoard)
	}
	boardCount := len(r.Boards)
	if boardCount > 0 {
		r.PinsPerBoard = r.Max / boardCount
	}
}

// Randomize will seed based on current time
func (r *Randomizer) Randomize() {
	rand.Seed(time.Now().UnixNano())
}

func trim(pins *[]Pin, limit int) {
	for len(*pins) > limit {
		*pins = remove(*pins, rand.Intn(len(*pins)))
	}
}

func remove(s []Pin, i int) []Pin {
	s[i] = s[len(s)-1]
	// We do not need to put s[i] at the end, as it will be discarded anyway
	return s[:len(s)-1]
}
