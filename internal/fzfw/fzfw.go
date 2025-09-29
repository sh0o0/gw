package fzfw

import (
	"errors"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/manifoldco/promptui"
)

var errNoItems = errors.New("no items")

// Select presents an interactive fuzzy picker backed by promptui.
func Select(prompt string, items []string) (string, error) {
	if len(items) == 0 {
		return "", errNoItems
	}

	label := strings.TrimSpace(prompt)
	selector := promptui.Select{
		Label:             label,
		Items:             items,
		Size:              promptSize(len(items)),
		StartInSearchMode: true,
		Searcher: func(input string, index int) bool {
			item := items[index]
			if input == "" {
				return true
			}
			return fuzzy.MatchNormalizedFold(input, item)
		},
	}

	_, result, err := selector.Run()
	if err != nil {
		if errors.Is(err, promptui.ErrInterrupt) || errors.Is(err, promptui.ErrEOF) {
			return "", nil
		}
		return "", err
	}

	return result, nil
}

func promptSize(count int) int {
	const maxSize = 12
	if count < maxSize {
		return count
	}
	return maxSize
}
