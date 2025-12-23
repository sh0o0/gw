package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/sh0o0/gw/internal/gitx"
	"github.com/spf13/cobra"
)

func newAICmd() *cobra.Command {
	var opts fuzzyDisplayOptions
	var aiCmd string
	cmd := &cobra.Command{
		Use:   "ai [branch]",
		Short: "Open worktree in AI CLI",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ai := resolveAI(aiCmd)
			if ai == "" {
				return errors.New("no AI CLI specified: use --ai flag or set gw.ai config")
			}
			if len(args) == 1 {
				return openAIForBranch(ai, args[0])
			}
			return openAIInteractive(ai, opts)
		},
	}
	cmd.Flags().StringVarP(&aiCmd, "ai", "a", "", "AI CLI command to use (default: gw.ai config)")
	cmd.Flags().BoolVar(&opts.showPath, "show-path", false, "display worktree path in fuzzy finder")
	return cmd
}

func resolveAI(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if v, err := gitx.ConfigGet("", configKeyAI); err == nil && v != "" {
		return v
	}
	return ""
}

func openAIInteractive(ai string, opts fuzzyDisplayOptions) error {
	wts, err := gitx.ListWorktrees("")
	if err != nil {
		return err
	}
	primaryPath, _ := primaryWorktreePath()
	entries := buildWorktreeEntries(wts, nil, primaryPath)
	if len(entries) == 0 {
		return errors.New("no worktrees available for selection")
	}
	collection := newWorktreeCollection(entries, opts)
	root, err := gitx.Root("")
	if err != nil {
		root = ""
	}
	resolver := gitx.NewBranchStatusResolver(root)
	startWorktreeStatusLoader(collection, resolver)
	idx, err := fuzzyfinder.Find(&collection.slice, func(i int) string {
		return collection.itemString(i)
	},
		fuzzyfinder.WithPromptString("Select worktree to open in AI CLI: "),
		fuzzyfinder.WithHotReloadLock(&collection.lock),
	)
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return errors.New("selection cancelled")
		}
		return err
	}
	entry, ok := collection.entryByIndex(idx)
	if !ok {
		return errors.New("selection cancelled")
	}
	return openAICmd(ai, entry.path)
}

func openAIForBranch(ai, branch string) error {
	p, err := gitx.FindWorktreeByBranch("", branch)
	if err != nil {
		return fmt.Errorf("no worktree found for branch: %s", branch)
	}
	return openAICmd(ai, p)
}

func openAICmd(ai, path string) error {
	cmd := exec.Command(ai)
	cmd.Dir = path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
