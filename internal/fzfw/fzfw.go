package fzfw

import (
	"errors"
	"sort"
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

// SelectMultiple presents an interactive fuzzy picker that allows multi-selection.
func SelectMultiple(prompt string, items []string) ([]string, error) {
	if len(items) == 0 {
		return nil, errNoItems
	}

	label := strings.TrimSpace(prompt)
	options := make([]fuzzyfinder.Option, 0, 1)
	if label != "" {
		options = append(options, fuzzyfinder.WithPromptString(label))
	}

	idxs, err := fuzzyfinder.FindMulti(
		items,
		func(i int) string {
			return items[i]
		},
		options...,
	)
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return nil, nil
		}
		return nil, err
	}
	if len(idxs) == 0 {
		return nil, nil
	}

	sort.Ints(idxs)

	selected := make([]string, 0, len(idxs))
	prev := -1
	for _, idx := range idxs {
		if idx == prev {
			continue
		}
		selected = append(selected, items[idx])
		prev = idx
	}

	return selected, nil
}
