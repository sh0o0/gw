package cli

import (
	"fmt"
	"strings"

	"github.com/sh0o0/gw/internal/gitx"
	"github.com/sh0o0/gw/internal/worktree"
	"github.com/spf13/cobra"
)

func newNewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new <branch>",
		Short: "Create new worktree with a new branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			branch := args[0]

			if path, err := gitx.FindWorktreeByBranch("", branch); err == nil && path != "" {
				return fmt.Errorf("worktree already exists for branch: %s", branch)
			}

			if exists, _ := gitx.BranchExists("", branch); exists {
				return fmt.Errorf("branch already exists: %s", branch)
			}

			prevRev, _ := gitx.Cmd("", "rev-parse", "--verify", "HEAD")
			prevRev = strings.TrimSpace(prevRev)
			prevBranch, _ := gitx.BranchAt(".")

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

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <branch>",
		Short: "Create new worktree for an existing branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			branch := args[0]

			gitx.Cmd("", "fetch", "origin", branch)

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
