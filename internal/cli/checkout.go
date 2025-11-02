package cli

import (
	"strings"

	"github.com/sh0o0/gw/internal/gitx"
	"github.com/sh0o0/gw/internal/worktree"
	"github.com/spf13/cobra"
)

func newCheckoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "checkout <branch>",
		Short:   "Switch to existing worktree or create new one for branch",
		Aliases: []string{"co"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prevRev, _ := gitx.Cmd("", "rev-parse", "--verify", "HEAD")
			prevRev = strings.TrimSpace(prevRev)
			prevBranch, _ := gitx.BranchAt(".")

			branch := args[0]
			path, err := gitx.FindWorktreeByBranch("", branch)
			if err == nil && path != "" {
				if err := switchToPath(path); err != nil {
					return err
				}
				newRev, _ := gitx.Cmd(path, "rev-parse", "--verify", "HEAD")
				newRev = strings.TrimSpace(newRev)
				newBranch, _ := gitx.BranchAt(path)
				runPostCheckoutWithCWD(prevRev, newRev, prevBranch, newBranch, path)
				return nil
			}

			p, err := worktree.ComputeWorktreePath("", branch)
			if err != nil {
				return err
			}
			if _, err := gitx.Cmd("", "worktree", "add", p, "-b", branch); err != nil {
				return err
			}
			if err := postCreateWorktree(p); err != nil {
				return err
			}

			newRev, _ := gitx.Cmd(p, "rev-parse", "--verify", "HEAD")
			newRev = strings.TrimSpace(newRev)
			newBranch, _ := gitx.BranchAt(p)
			runPostCheckoutWithCWD(prevRev, newRev, prevBranch, newBranch, p)
			return nil
		},
	}
}

func newRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <branch>",
		Short: "Create new worktree for specified branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			branch := args[0]
			p, err := worktree.ComputeWorktreePath("", branch)
			if err != nil {
				return err
			}
			if _, err := gitx.Cmd("", "worktree", "add", p, branch); err != nil {
				return err
			}
			return postCreateWorktree(p)
		},
	}
}
