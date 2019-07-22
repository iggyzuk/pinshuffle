package controllers

import (
	"net/url"
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

// ParseLimit returns max value from the URL
func ParseLimit(URL *url.URL) int32 {
	return 100
}
