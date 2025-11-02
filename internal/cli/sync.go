package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/sh0o0/gw/internal/gitx"
	"github.com/sh0o0/gw/internal/worktree"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync symlinks from primary worktree to current worktree",
		RunE: func(cmd *cobra.Command, args []string) error {
			current, err := gitx.CurrentWorktreePath("")
			if err != nil {
				return err
			}
			primary, err := primaryWorktreePath()
			if err != nil {
				return err
			}
			if current == primary {
				return errors.New("you are in the primary worktree; nothing to sync")
			}
			count, err := worktree.CreateSymlinksFromGitignored(primary, current)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Synced %d symlink(s)\n", count)
			return nil
		},
	}
}
