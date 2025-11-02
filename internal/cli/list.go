package cli

import (
	"fmt"

	"github.com/sh0o0/gw/internal/gitx"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all worktrees",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := gitx.Cmd("", "worktree", "list")
			if err != nil {
				return err
			}
			fmt.Print(out)
			return nil
		},
	}
}

func newPruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Prune worktrees",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := gitx.Cmd("", "worktree", "prune")
			if err != nil {
				return err
			}
			fmt.Print(out)
			return nil
		},
	}
}
