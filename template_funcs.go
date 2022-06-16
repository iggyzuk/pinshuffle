package main

import (
	"sort"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func IsBoardSelected(boards []string, id string) bool {
	return slices.Contains(boards, id)
}

func SortBoards(boards map[string]*TemplateBoard) []*TemplateBoard {
	list := maps.Values(boards)

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})

	return list
}

func Iterate(count int) []int {
	var i int
	var Items []int
	for i = 0; i < (count); i++ {
		Items = append(Items, i)
	}
	return Items
}
