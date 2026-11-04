package main

import (
	"slices"
	"sort"
)

func IsBoardSelected(boards []string, id string) bool {
	return slices.Contains(boards, id)
}

func SortBoards(boards map[string]*TemplateBoard) []*TemplateBoard {
	list := make([]*TemplateBoard, 0, len(boards))
	for _, board := range boards {
		list = append(list, board)
	}

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})

	return list
}
