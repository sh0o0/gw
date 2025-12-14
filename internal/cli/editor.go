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

func newEditorCmd() *cobra.Command {
	var opts fuzzyDisplayOptions
	var editorCmd string
	cmd := &cobra.Command{
		Use:     "editor [branch]",
		Short:   "Open worktree in editor",
		Aliases: []string{"ed"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			editor := resolveEditor(editorCmd)
			if editor == "" {
				return errors.New("no editor specified: use --editor flag or set $EDITOR environment variable")
			}
			if len(args) == 1 {
				return openEditorForBranch(editor, args[0])
			}
			return openEditorInteractive(editor, opts)
		},
	}
	cmd.Flags().StringVarP(&editorCmd, "editor", "e", "", "editor command to use (default: $EDITOR)")
	cmd.Flags().BoolVar(&opts.showPath, "show-path", false, "display worktree path in fuzzy finder")
	return cmd
}

func resolveEditor(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if v, err := gitx.ConfigGet("", "gw.editor"); err == nil && v != "" {
		return v
	}
	return os.Getenv("EDITOR")
}

func openEditorInteractive(editor string, opts fuzzyDisplayOptions) error {
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
		fuzzyfinder.WithPromptString("Select worktree to open in editor: "),
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
	return openEditorCmd(editor, entry.path)
}

func openEditorForBranch(editor, branch string) error {
	p, err := gitx.FindWorktreeByBranch("", branch)
	if err != nil {
		return fmt.Errorf("no worktree found for branch: %s", branch)
	}
	return openEditorCmd(editor, p)
}

func openEditorCmd(editor, path string) error {
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
