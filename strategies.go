package main

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

// Upper limit for loading the user's account while they wait for the page.
const requestTimeout = 30 * time.Second

type StrategyFunc func(c fiber.Ctx) (TemplateModel, error)

// SectionsFunc lists the sections of one of the user's boards.
type SectionsFunc func(c fiber.Ctx, boardId string) ([]Section, error)

func GetMockSections(c fiber.Ctx, boardId string) ([]Section, error) {
	return mockSections(boardId), nil
}

func GetSections(c fiber.Ctx, boardId string) ([]Section, error) {
	tokens := readTokens(c)
	if tokens.AccessToken == "" {
		return nil, ErrUnauthorized
	}

	ctx, cancel := context.WithTimeout(c.Context(), requestTimeout)
	defer cancel()

	return app.PinClient.WithToken(tokens.AccessToken).FetchSections(ctx, boardId)
}

func GetMockTemplateModel(c fiber.Ctx) (TemplateModel, error) {
	tmplController := NewTemplateController()
	tmplController.Mock(c.RequestCtx().URI())

	followTask(c, tmplController, func(key string) *Task {
		query := tmplController.Model.UrlQuery

		expected := 0
		if len(query.Sections) == 0 {
			expected = len(query.Boards) * mockPagesPerTask * mockPinsPerPage
		}

		task := app.Tasks.New(key)
		task.Describe(query, expected)
		go task.Simulate()
		return task
	})

	if !tmplController.Model.Loading {
		tmplController.AddMockPins()
	}

	return tmplController.Model, nil
}

func GetTemplateModel(c fiber.Ctx) (TemplateModel, error) {
	tmplController := NewTemplateController()
	model := &tmplController.Model

	tokens := readTokens(c)

	if tokens.Empty() {
		showLoggedOut(c, model, "Waiting for access to Pinterest account")
		return *model, nil
	}

	model.Authenticated = true

	ctx, cancel := context.WithTimeout(c.Context(), requestTimeout)
	defer cancel()

	client, account, err := loadAccount(ctx, c, tokens)

	if errors.Is(err, ErrUnauthorized) {
		log.Printf("Session ended: %v", err)
		clearTokens(c)
		showLoggedOut(c, model, "Your Pinterest session has expired, please sign in again")
		return *model, nil
	}
	if err != nil {
		model.Error = err.Error()
	}

	if account.User != nil {
		model.User = TemplateUser{
			Name:    account.User.Username,
			IconURL: account.User.ProfileImage,
			URL:     account.User.WebsiteURL,
		}
	}

	// Without the boards the URL's selection can't be validated. Keep any running
	// task for the next load, Pinterest might just be having a moment.
	if len(account.Boards) == 0 {
		if err == nil {
			model.Message = "No boards found"
		}
		return *model, nil
	}

	clientBoards := make(map[string]Board, len(account.Boards))
	for _, board := range account.Boards {
		clientBoards[board.Id] = board
	}

	if parseErr := tmplController.ParseUrlQueries(c.RequestCtx().URI(), clientBoards); parseErr != nil {
		model.Error = parseErr.Error()
	}

	if err := tmplController.LoadSections(ctx, client.FetchSections); err != nil {
		model.Error = err.Error()
		return *model, nil
	}

	task := followTask(c, tmplController, func(key string) *Task {
		query := model.UrlQuery

		// Pinterest doesn't say how many pins are in a section.
		expected := 0
		if len(query.Sections) == 0 {
			for _, boardId := range query.Boards {
				expected += clientBoards[boardId].PinCount
			}
		}

		task := app.Tasks.New(key)
		task.Describe(query, expected)

		// Runs after this response is sent, reloading the page when it's done.
		go task.Process(client, query)

		return task
	})

	if task != nil {
		for i := range task.Pins {
			tmplController.AddPin(&task.Pins[i])
		}
		if task.Error != "" {
			model.Error = task.Error
		} else if len(model.Pins) == 0 {
			model.Message = "Nothing to show from what you selected"
		}
	}

	return *model, nil
}

// followTask connects the page to the user's background task. It returns the
// task once that's complete, and is done with it. If there's none yet for what's
// selected then start makes one. Meanwhile the model is set to show it loading.
func followTask(c fiber.Ctx, tmplController *TemplateController, start func(key string) *Task) *Task {
	model := &tmplController.Model
	key := taskKey(model.UrlQuery)

	if taskId := c.Cookies(taskCookie); taskId != "" {
		task, exists := app.Tasks.Get(taskId)

		switch {
		case exists && task.Key == key && task.IsComplete():
			app.Tasks.Delete(task.Id)
			clearCookie(c, taskCookie)
			return task

		case exists && task.Key == key:
			model.Loading = true
			model.TaskId = task.Id
			return nil
		}

		// The task is gone (the server restarted) or was for a different
		// selection, so start over below.
		if exists {
			app.Tasks.Delete(task.Id)
		}
		clearCookie(c, taskCookie)
	}

	if len(model.UrlQuery.Sources()) > 0 {
		task := start(key)
		c.Cookie(sessionCookie(taskCookie, task.Id, time.Now().Add(taskTTL)))

		model.Loading = true
		model.TaskId = task.Id
	}

	return nil
}

func showLoggedOut(c fiber.Ctx, model *TemplateModel, message string) {
	model.Authenticated = false
	model.Message = message

	if c.Query("auth") == "failed" {
		model.Message = ""
		model.Error = "Could not sign in with Pinterest, please try again."
	}
}

type Account struct {
	User   *UserAccount
	Boards []Board
}

// loadAccount fetches the user's account as a client holding valid tokens. The
// access token is renewed when it's about to expire, or if Pinterest turns it
// down anyway. Renewed tokens are written to the response cookies.
//
// ErrUnauthorized means the session can't be saved and the user has to log in.
// Any other error can come with a partially filled account.
func loadAccount(ctx context.Context, c fiber.Ctx, tokens Tokens) (*PinterestClient, Account, error) {
	refreshed := false

	if tokens.NeedsRefresh(time.Now()) {
		fresh, err := app.PinClient.RefreshTokens(ctx, tokens)

		switch {
		case err == nil:
			tokens = fresh
			refreshed = true
			writeTokens(c, tokens)
		case tokens.Expired(time.Now()):
			return nil, Account{}, err
		default:
			// The current token still works. Next request tries again.
			log.Printf("Early token refresh failed: %v", err)
		}
	}

	client := app.PinClient.WithToken(tokens.AccessToken)
	account, err := fetchAccount(ctx, client)

	if errors.Is(err, ErrUnauthorized) && !refreshed && tokens.RefreshToken != "" {
		fresh, refreshErr := app.PinClient.RefreshTokens(ctx, tokens)
		if refreshErr != nil {
			return nil, Account{}, refreshErr
		}
		writeTokens(c, fresh)

		client = app.PinClient.WithToken(fresh.AccessToken)
		account, err = fetchAccount(ctx, client)
	}

	return client, account, err
}

func fetchAccount(ctx context.Context, client *PinterestClient) (Account, error) {
	var account Account
	var userErr, boardsErr error
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		account.User, userErr = client.FetchUserAccount(ctx)
	}()
	go func() {
		defer wg.Done()
		account.Boards, boardsErr = client.FetchBoards(ctx)
	}()
	wg.Wait()

	return account, errors.Join(userErr, boardsErr)
}
