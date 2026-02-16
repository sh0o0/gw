package cli

import (
	"errors"

	"github.com/sh0o0/gw/internal/gitx"
	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	var verbose bool
	var hookBackground bool
	var hookForeground bool
	var noHooks bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Run post-create setup (symlinks + hooks) on current worktree",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			effectiveHookBg := hookBackground || (cfg.HooksBackground && !hookForeground)

			current, err := gitx.CurrentWorktreePath("")
			if err != nil {
				return err
			}
			primary, err := primaryWorktreePath()
			if err != nil {
				return err
			}
			if samePath(current, primary) {
				return errors.New("you are in the primary worktree; setup is for secondary worktrees")
			}

			if err := createSymlinks(current, PostCreateOptions{Verbose: verbose}); err != nil {
				return err
			}

			if !noHooks {
				branch, err := gitx.BranchAt(".")
				if err != nil {
					return err
				}
				runPostCreate(branch, current, effectiveHookBg)
			}

			out.Success("Setup complete")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show each symlink created")
	cmd.Flags().BoolVar(&hookBackground, "hook-bg", false, "Run post-create hook in background")
	cmd.Flags().BoolVar(&hookForeground, "hook-fg", false, "Run post-create hook in foreground (override config)")
	cmd.Flags().BoolVar(&noHooks, "no-hooks", false, "Skip post-create hooks")
	cmd.MarkFlagsMutuallyExclusive("hook-bg", "hook-fg", "no-hooks")

	return cmd
}
