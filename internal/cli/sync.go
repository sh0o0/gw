package cli

import (
	"errors"

	"github.com/sh0o0/gw/internal/gitx"
	"github.com/sh0o0/gw/internal/worktree"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
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
			opts := worktree.SymlinkOptions{Verbose: verbose}
			count, err := worktree.CreateSymlinksFromGitignored(primary, current, opts)
			if err != nil {
				return err
			}
			out.Link("Synced %d symlink(s)", count)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show each symlink created")
	return cmd
}
