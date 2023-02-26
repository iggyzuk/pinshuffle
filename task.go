package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	Id         string
	IsComplete bool
	Timestamp  time.Time
	Error      string
	Pins       []Pin
}

func NewTask() *Task {

	task := &Task{
		Id:         uuid.Must(uuid.NewRandom()).String(),
		IsComplete: false,
		Timestamp:  time.Now(),
		Pins:       nil,
	}

	app.Tasks[task.Id] = task

	fmt.Println("New Task: " + task.Id)

	return task
}

func (task *Task) Process(pinClient *PinterestClient, clientBoards map[string]Board, query TemplateUrlQuery) {

	// Make sure there are boards to process
	if len(query.Boards) <= 0 {
		task.IsComplete = true
		return
	}

	// Start the randomizer, it will load all pins from every selected board and mix them up
	randomizer := NewRandomizer(pinClient, clientBoards)
	randomizedPins, err := randomizer.GetRandomizedPins(query.Max, query.Boards)

	if err != nil {
		task.Error = err.Error()
	}

	// Save the results and complete
	task.Pins = randomizedPins
	task.IsComplete = true

	fmt.Println("Task Complete: " + task.Id)

	// We are now ready to be consumed
}
