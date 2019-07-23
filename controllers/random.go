package controllers

import (
	"math/rand"
	"time"

	"iggyzuk.com/shuffle/models"
)

// Randomize will seed based on current time
func Randomize() {
	rand.Seed(time.Now().UnixNano())
}

func trim(pins *[]models.Pin, limit int) {
	for len(*pins) > limit {
		*pins = remove(*pins, rand.Intn(len(*pins)))
	}
}

func remove(s []models.Pin, i int) []models.Pin {
	s[i] = s[len(s)-1]
	// We do not need to put s[i] at the end, as it will be discarded anyway
	return s[:len(s)-1]
}
