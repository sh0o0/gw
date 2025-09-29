package fzfw

import (
	"errors"
	"strings"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
)

var errNoItems = errors.New("no items")

// Select presents an interactive fuzzy picker backed by go-fuzzyfinder.
func Select(prompt string, items []string) (string, error) {
	if len(items) == 0 {
		return "", errNoItems
	}

	label := strings.TrimSpace(prompt)
	options := make([]fuzzyfinder.Option, 0, 1)
	if label != "" {
		options = append(options, fuzzyfinder.WithPromptString(label))
	}

	idx, err := fuzzyfinder.Find(
		items,
		func(i int) string {
			return items[i]
		},
		options...,
	)
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return "", nil
		}
		return "", err
	}

	return items[idx], nil
}
