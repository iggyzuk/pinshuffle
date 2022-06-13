package main

import "golang.org/x/exp/slices"

func IsBoardSelected(boards []string, id string) bool {
	return slices.Contains(boards, id)
}

func Iterate(count int) []int {
	var i int
	var Items []int
	for i = 0; i < (count); i++ {
		Items = append(Items, i)
	}
	return Items
}
