package main

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const (
	// A task gets a bit less time to run than it's kept around for, so that the
	// client always hears how it ended.
	taskTimeout = 4 * time.Minute
	taskTTL     = 5 * time.Minute

	// How long the fake task of mock mode takes.
	mockTaskDuration = 4 * time.Second
	mockPagesPerTask = 3
	mockPinsPerPage  = 40
)

// Task is a background fetch of a user's pins. Pins and Error must only be read
// once IsComplete returns true.
type Task struct {
	Id        string
	Key       string
	Timestamp time.Time
	Error     string
	Pins      []Pin

	total    int
	unit     string
	expected int

	sourcesDone atomic.Int64
	pinsLoaded  atomic.Int64

	done chan struct{}
}

// TaskProgress is what the loading screen shows.
type TaskProgress struct {
	Status   string `json:"status"`
	Percent  int    `json:"percent"`
	Done     int    `json:"done"`
	Total    int    `json:"total"`
	Unit     string `json:"unit"`
	Pins     int    `json:"pins"`
	Expected int    `json:"expected"`
}

func (task *Task) IsComplete() bool {
	select {
	case <-task.done:
		return true
	default:
		return false
	}
}

// Describe sets what progress is measured against, before the task starts.
// expected is the number of pins to load, or 0 if that isn't known: Pinterest
// counts the pins of a board but not of a section.
func (task *Task) Describe(query TemplateUrlQuery, expected int) {
	task.total = len(query.Sources())
	task.expected = expected

	switch {
	case len(query.Sections) == 0:
		task.unit = "boards"
	case len(query.Boards) == 0:
		task.unit = "sections"
	default:
		task.unit = "boards and sections"
	}
}

func (task *Task) AddPins(count int) {
	task.pinsLoaded.Add(int64(count))
}

func (task *Task) SourceDone() {
	task.sourcesDone.Add(1)
}

func (task *Task) Progress() TaskProgress {
	progress := TaskProgress{
		Status:   "processing",
		Done:     int(task.sourcesDone.Load()),
		Total:    task.total,
		Unit:     task.unit,
		Pins:     int(task.pinsLoaded.Load()),
		Expected: task.expected,
	}

	if task.IsComplete() {
		progress.Status = "complete"
		progress.Percent = 100
		return progress
	}

	// Going by pins moves steadily, even through one enormous board. Pinterest's
	// count can be off though, so finished boards count for at least their share.
	// It stays below 100 as the pins aren't ready before the task is complete.
	if progress.Total > 0 {
		progress.Percent = progress.Done * 100 / progress.Total
	}
	if progress.Expected > 0 {
		progress.Percent = max(progress.Percent, progress.Pins*100/progress.Expected)
	}
	progress.Percent = min(progress.Percent, 99)

	return progress
}

// Process runs the task. pinClient must be the requesting user's client.
func (task *Task) Process(pinClient *PinterestClient, query TemplateUrlQuery) {
	defer close(task.done)

	// An unexpected panic in a goroutine would take down the server for everyone.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Task %s panicked: %v", task.Id, r)
			task.Error = "Something went wrong while shuffling your pins."
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), taskTimeout)
	defer cancel()

	randomizer := NewRandomizer(pinClient, task)
	pins, err := randomizer.GetRandomizedPins(ctx, query.Max, query.Sources())

	if err != nil {
		task.Error = err.Error()
	}
	task.Pins = pins
}

// Simulate acts out a task for mock mode, which has no Pinterest to wait for.
func (task *Task) Simulate() {
	defer close(task.done)

	step := mockTaskDuration / time.Duration(max(task.total, 1)*mockPagesPerTask)

	for range task.total {
		for range mockPagesPerTask {
			time.Sleep(step)
			task.AddPins(mockPinsPerPage)
		}
		task.SourceDone()
	}
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
	sources := make([]string, 0, len(query.Boards)+len(query.Sections))
	for _, source := range query.Sources() {
		sources = append(sources, source.BoardId+":"+source.SectionId)
	}
	slices.Sort(sources)
	return fmt.Sprintf("%d|%d|%s", query.Max, query.ImageResolution, strings.Join(sources, ","))
}
