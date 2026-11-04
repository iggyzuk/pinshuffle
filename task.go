package main

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// A task gets a bit less time to run than it's kept around for, so that the
	// client always hears how it ended.
	taskTimeout = 4 * time.Minute
	taskTTL     = 5 * time.Minute
)

// Task is a background fetch of a user's pins. Pins and Error must only be read
// once IsComplete returns true.
type Task struct {
	Id        string
	Key       string
	Timestamp time.Time
	Error     string
	Pins      []Pin

	done chan struct{}
}

func (task *Task) IsComplete() bool {
	select {
	case <-task.done:
		return true
	default:
		return false
	}
}

// Process runs the task. pinClient must be the requesting user's client.
func (task *Task) Process(pinClient *PinterestClient, clientBoards map[string]Board, query TemplateUrlQuery) {
	defer close(task.done)

	// An unexpected panic in a goroutine would take down the server for everyone.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Task %s panicked: %v", task.Id, r)
			task.Error = "⚠️ Something went wrong while shuffling your pins."
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), taskTimeout)
	defer cancel()

	randomizer := NewRandomizer(pinClient, clientBoards)
	pins, err := randomizer.GetRandomizedPins(ctx, query.Max, query.Boards)

	if err != nil {
		task.Error = err.Error()
	}
	task.Pins = pins
}

// TaskStore is safe for concurrent use, since every request can touch it.
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

func NewTaskStore() *TaskStore {
	return &TaskStore{tasks: make(map[string]*Task)}
}

func (store *TaskStore) New(key string) *Task {
	task := &Task{
		Id:        uuid.NewString(),
		Key:       key,
		Timestamp: time.Now(),
		done:      make(chan struct{}),
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	store.tasks[task.Id] = task

	return task
}

func (store *TaskStore) Get(id string) (*Task, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	task, exists := store.tasks[id]
	return task, exists
}

func (store *TaskStore) Delete(id string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.tasks, id)
}

// DeleteExpired drops tasks whose users never came back for the results.
func (store *TaskStore) DeleteExpired(now time.Time) {
	store.mu.Lock()
	defer store.mu.Unlock()

	for id, task := range store.tasks {
		if now.Sub(task.Timestamp) >= taskTTL {
			delete(store.tasks, id)
		}
	}
}

// taskKey identifies what a task was started for, so its results are only shown
// on the page that asked for them.
func taskKey(query TemplateUrlQuery) string {
	boards := slices.Clone(query.Boards)
	slices.Sort(boards)
	return fmt.Sprintf("%d|%d|%s", query.Max, query.ImageResolution, strings.Join(boards, ","))
}
