package controllers

import (
	"fmt"
	"net/url"
	"strconv"
)

// ParseBoards returns a slice with board names from the URL
func ParseBoards(URL *url.URL) []string {
	bs, ok := URL.Query()["b"]

	if !ok {
		return []string{}
	}

	var boards []string

	for _, b := range bs {
		boards = append(boards, b)
	}

	return boards
}

// ParseMax returns the max number of pins from URL
func ParseMax(URL *url.URL) int32 {
	maxes, ok := URL.Query()["max"]

	if !ok {
		return 10000
	}

	i, err := strconv.ParseInt(maxes[0], 10, 32)
	if err != nil {
		fmt.Printf("Could not parse max")
		panic(err)
	}

	result := int32(i)

	if result < 1 {
		result = 1
	}

	if result > 1000 {
		result = 1000
	}

	return result
}
