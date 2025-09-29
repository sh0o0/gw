package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "gw",
		Short:         "Git worktree power tool",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(
		newLinkCmd(),
		newUnlinkCmd(),
		newSwitchCmd(),
		newCheckoutCmd(),
		newRestoreCmd(),
		newListCmd(),
		newPruneCmd(),
		newRemoveCmd(),
	)

	return cmd
}
