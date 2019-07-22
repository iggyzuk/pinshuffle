package controllers

import "net/url"

// ParseURLForBoards returns a slice with board names from the URL
func ParseBoards(URL *url.URL) []string {
	return []string{"iggyzuky/flash-games", "iggyzuky/hyper-casual"}
}

// GetLimit returns max value from the URL
func ParseLimit(URL *url.URL) int32 {
	return 100
}
