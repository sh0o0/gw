package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sh0o0/gw/internal/fsutil"
	"github.com/sh0o0/gw/internal/gitx"
	"github.com/sh0o0/gw/internal/worktree"
	"github.com/spf13/cobra"
)

func newMvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mv <old-branch> <new-branch>",
		Short: "Rename a branch and relocate its worktree directory",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return moveWorktree(args[0], args[1])
		},
	}
}

func moveWorktree(oldBranch, newBranch string) error {
	if oldBranch == "" || newBranch == "" {
		return errors.New("both branch names required")
	}
	if oldBranch == newBranch {
		return errors.New("branch names must differ")
	}
	oldPath, err := gitx.FindWorktreeByBranch("", oldBranch)
	if err != nil {
		return fmt.Errorf("no worktree found for branch: %s", oldBranch)
	}
	oldPath = filepath.Clean(oldPath)

	sameDir := func(p1, p2 string) bool {
		if p1 == p2 {
			return true
		}
		info1, err1 := os.Stat(p1)
		info2, err2 := os.Stat(p2)
		if err1 == nil && err2 == nil && os.SameFile(info1, info2) {
			return true
		}
		r1, err1 := filepath.EvalSymlinks(p1)
		r2, err2 := filepath.EvalSymlinks(p2)
		return err1 == nil && err2 == nil && r1 == r2
	}

	if primary, err := primaryWorktreePath(); err == nil {
		if sameDir(primary, oldPath) {
			return errors.New("cannot move primary worktree")
		}
	}

	resolvedBranch, err := gitx.BranchAt(oldPath)
	if err != nil {
		return err
	}
	if resolvedBranch == "" || resolvedBranch == "HEAD" {
		return errors.New("cannot move detached worktree")
	}
	if resolvedBranch != oldBranch {
		return fmt.Errorf("worktree branch mismatch: expected %s, got %s", oldBranch, resolvedBranch)
	}

	destPath, err := worktree.ComputeWorktreePath(oldPath, newBranch)
	if err != nil {
		return err
	}
	destPath = filepath.Clean(destPath)
	samePath := sameDir(destPath, oldPath)

	if !samePath {
		if err := fsutil.EnsureDir(filepath.Dir(destPath)); err != nil {
			return err
		}
		if _, statErr := os.Stat(destPath); statErr == nil {
			return fmt.Errorf("destination already exists: %s", destPath)
		} else if !errors.Is(statErr, fs.ErrNotExist) {
			return statErr
		}
	}

	caller, err := callerCWD()
	if err != nil {
		return err
	}
	caller = filepath.Clean(caller)
	inOld := caller == oldPath || strings.HasPrefix(caller, oldPath+string(os.PathSeparator))
	relWithin := "."
	if inOld {
		if rel, relErr := filepath.Rel(oldPath, caller); relErr == nil {
			relWithin = rel
		}
	}

	if _, err := gitx.Cmd(oldPath, "branch", "-m", newBranch); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Renamed branch: %s -> %s\n", oldBranch, newBranch)

	if !samePath {
		if _, err := gitx.Cmd("", "worktree", "move", oldPath, destPath); err != nil {
			if _, revertErr := gitx.Cmd(oldPath, "branch", "-m", oldBranch); revertErr == nil {
				fmt.Fprintln(os.Stderr, "Reverted branch rename because worktree move failed")
			}
			return err
		}
		fmt.Fprintf(os.Stderr, "Moved worktree: %s -> %s\n", oldPath, destPath)
	} else {
		destPath = oldPath
	}

	printPath := destPath
	if inOld && relWithin != "." {
		candidate := filepath.Join(destPath, relWithin)
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			printPath = candidate
		}
	}
	fmt.Println(printPath)
	return nil
}
