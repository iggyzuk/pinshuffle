package controllers

import (
	"fmt"
	"log"

	"github.com/carrot/go-pinterest"
	"github.com/carrot/go-pinterest/controllers"
	"iggyzuk.com/shuffle/models"
)

// GetBoardsPins returns a pointer to a slice contains ALL pins from ALL the boards
func GetBoardsPins(client *pinterest.Client, boards []string, accessToken string) (*[]models.Pin, error) {
	var allPins []models.Pin

	for _, board := range boards {
		log.Println("Board: " + board)

		pins, err := getBoardPins(
			client,
			board,
			accessToken,
		)

		if err != nil {
			return nil, err
		}

		allPins = append(allPins, *pins...)
	}
	return &allPins, nil
}

// getBoardPins returns a pointer to a slice containing ALL pins within a board
func getBoardPins(client *pinterest.Client, board string, accessToken string) (*[]models.Pin, error) {
	log.Println("Loading pins from: " + board)

	var boardPins []models.Pin

	pins, cursor, err := getPagePins(client, board, "")

	if err != nil {
		return nil, err
	}

	boardPins = append(boardPins, *pins...)

	for cursor != "" {
		pins, cursor, err = getPagePins(client, board, cursor)
		boardPins = append(boardPins, *pins...)

		if err != nil {
			return nil, err
		}
	}

	return &boardPins, nil
}

// getPagePins returns a pointer to a slice containing 100 (max) pins on a page of a board
func getPagePins(client *pinterest.Client, board string, cursor string) (*[]models.Pin, string, error) {
	var pagePins []models.Pin

	pins, page, err := client.Boards.Pins.Fetch(
		board,
		&controllers.BoardsPinsFetchOptionals{
			Cursor: cursor,
			Limit:  100,
		},
	)

	if err != nil {
		fmt.Println(err)
		return nil, "", err
	}

	for _, pin := range *pins {
		pagePins = append(pagePins, models.Pin{
			ImageURL: pin.Image.Original.Url,
			PinURL:   pin.Url,
			Color:    pin.Color,
		})
	}

	return &pagePins, page.Cursor, nil
}
