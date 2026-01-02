package cli

import (
	"fmt"

	"github.com/sh0o0/gw/internal/tui"
	"github.com/spf13/cobra"
)

func newTuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI mode",
		Long:  "Launch a lazygit-style interactive TUI for managing git worktrees",
		RunE: func(cmd *cobra.Command, args []string) error {
			selectedPath, err := tui.Run()
			if err != nil {
				return err
			}
			if selectedPath != "" {
				fmt.Println(selectedPath)
			}
			return nil
		},
	}
}
