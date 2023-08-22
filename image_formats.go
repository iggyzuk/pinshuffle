package main

import (
	"net/http"
	"strings"
)

type ImageFormatResult struct {
	Url     string
	Success bool
}

type ImageFormatError string

func (err ImageFormatError) Error() string {
	return string(err)
}

// We know that original images follow the same structure .../150x150/abcdef.jpg
// The only problem is that we don't know the original file extension (jpg, png, gif)
// So we send 3 http requests and take the first one that succeeds.
func TryDifferentImageFormats(url string) (string, error) {

	urlOriginal := strings.Replace(url, "150x150", "originals", -1)

	urls := []string{
		urlOriginal,
		strings.Replace(urlOriginal, ".jpg", ".png", 1),
		strings.Replace(urlOriginal, ".jpg", ".gif", 1),
		strings.Replace(urlOriginal, ".jpg", ".webp", 1),
	}

	resultChan := make(chan (ImageFormatResult))

	for _, url := range urls {
		go checkUrl(url, resultChan)
	}

	for i := 0; i < len(urls); i++ {
		result := <-resultChan
		if result.Success {
			return result.Url, nil
		}
	}

	return "", ImageFormatError("Could not find any original urls")
}

func checkUrl(url string, ch chan ImageFormatResult) {
	resp, err := http.Get(url)

	result := ImageFormatResult{url, false}

	if err != nil {
		ch <- result
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		result.Success = true
	}

	ch <- result
}
