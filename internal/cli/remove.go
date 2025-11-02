package cli

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sh0o0/gw/internal/gitx"
	"github.com/spf13/cobra"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
)

func newRemoveCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     "remove [--force] [branch ...]",
		Short:   "Remove worktree(s) by fuzzy select or by branch names",
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				success := 0
				var failed []string
				for _, b := range args {
					if err := removeWorktreeByBranch(b, force); err != nil {
						failed = append(failed, b)
						fmt.Fprintf(os.Stderr, "✗ Failed to remove worktree for branch: %s\n\n", b)
					} else {
						success++
						fmt.Fprintf(os.Stderr, "✓ Successfully removed worktree for branch: %s\n\n", b)
					}
				}
				fmt.Fprintf(os.Stderr, "Summary:\n  Successfully removed: %d worktree(s)\n", success)
				if len(failed) > 0 {
					return fmt.Errorf("failed branches: %s", strings.Join(failed, ", "))
				}
				return nil
			}
			return removeInteractive(force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "force remove")
	return cmd
}

func removeWorktreeAtPath(path string, force bool) error {
	br, _ := gitx.BranchAt(path)
	fmt.Fprintf(os.Stderr, "Removing worktree: %s\n", path)
	if force {
		if _, err := gitx.Cmd("", "worktree", "remove", "--force", path); err != nil {
			return err
		}
	} else {
		if _, err := gitx.Cmd("", "worktree", "remove", path); err != nil {
			return err
		}
	}
	if br != "" && br != "HEAD" {
		fmt.Fprintf(os.Stderr, "Deleting branch: %s\n", br)
		if _, err := gitx.Cmd("", "branch", "-D", br); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to delete branch: %s\n", br)
		} else {
			fmt.Fprintf(os.Stderr, "Successfully deleted branch: %s\n", br)
		}
	}
	return nil
}

func removeInteractive(force bool) error {
	wts, err := gitx.ListWorktrees("")
	if err != nil {
		return err
	}
	current, _ := gitx.CurrentWorktreePath("")
	entries := buildWorktreeEntries(wts, func(wt gitx.Worktree) bool {
		return wt.Path == current
	})
	if len(entries) == 0 {
		return errors.New("no worktrees available for selection")
	}
	collection := newWorktreeCollection(entries)
	root, err := gitx.Root("")
	if err != nil {
		root = ""
	}
	resolver := gitx.NewBranchStatusResolver(root)
	startWorktreeStatusLoader(collection, resolver)
	idxs, err := fuzzyfinder.FindMulti(&collection.slice, func(i int) string {
		return collection.itemString(i)
	},
		fuzzyfinder.WithPromptString("Select worktree(s) to remove (TAB to mark, ENTER to confirm):"),
		fuzzyfinder.WithHotReloadLock(&collection.lock),
	)
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return errors.New("selection cancelled")
		}
		return err
	}
	if len(idxs) == 0 {
		return errors.New("selection cancelled")
	}
	selectedIdx := make([]int, 0, len(idxs))
	seen := make(map[int]struct{}, len(idxs))
	for _, idx := range idxs {
		if entryIdx, ok := collection.baseIndex(idx); ok {
			if _, dup := seen[entryIdx]; dup {
				continue
			}
			seen[entryIdx] = struct{}{}
			selectedIdx = append(selectedIdx, entryIdx)
		}
	}
	if len(selectedIdx) == 0 {
		return errors.New("selection cancelled")
	}
	sort.Ints(selectedIdx)
	success := 0
	var failed []string
	for _, idx := range selectedIdx {
		e := collection.base[idx]
		path := e.path
		if err := removeWorktreeAtPath(path, force); err != nil {
			failed = append(failed, path)
			fmt.Fprintf(os.Stderr, "✗ Failed to remove worktree: %s\n  %v\n\n", path, err)
			continue
		}
		success++
		fmt.Fprintf(os.Stderr, "✓ Successfully removed worktree: %s\n\n", path)
	}
	fmt.Fprintf(os.Stderr, "Summary:\n  Successfully removed: %d worktree(s)\n", success)
	if len(failed) > 0 {
		return fmt.Errorf("failed worktrees: %s", strings.Join(failed, ", "))
	}
	return nil
}

func removeWorktreeByBranch(branch string, force bool) error {
	p, err := gitx.FindWorktreeByBranch("", branch)
	if err != nil {
		return fmt.Errorf("no worktree found for branch: %s", branch)
	}
	current, _ := gitx.CurrentWorktreePath("")
	if current == p {
		return fmt.Errorf("cannot remove current worktree for branch: %s", branch)
	}
	return removeWorktreeAtPath(p, force)
}
