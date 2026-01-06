package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/sh0o0/gw/internal/gitx"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var opts fuzzyDisplayOptions
	cmd := &cobra.Command{
		Use:   "run [branch] -- <command> [args...]",
		Short: "Run command in a specific worktree",
		Long: `Run a command in a specific worktree directory.

If branch is not specified, an interactive fuzzy finder will be shown to select the worktree.
The command and its arguments should be specified after '--'.

Examples:
  gw run -- npm install
  gw run feature/foo -- make build
  gw run main -- go test ./...`,
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			dashIdx := cmd.ArgsLenAtDash()
			if dashIdx == -1 {
				return errors.New("command required: use 'gw run [branch] -- <command> [args...]'")
			}

			var branch string
			if dashIdx > 0 {
				branch = args[0]
			}

			cmdArgs := args[dashIdx:]
			if len(cmdArgs) == 0 {
				return errors.New("command required after '--'")
			}

			if branch != "" {
				return runCmdForBranch(branch, cmdArgs)
			}
			return runCmdInteractive(cmdArgs, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.showPath, "show-path", false, "display worktree path in fuzzy finder")
	return cmd
}

func runCmdInteractive(cmdArgs []string, opts fuzzyDisplayOptions) error {
	wts, err := gitx.ListWorktrees("")
	if err != nil {
		return err
	}
	primaryPath, _ := primaryWorktreePath()
	entries := buildWorktreeEntries(wts, nil, primaryPath)
	if len(entries) == 0 {
		return errors.New("no worktrees available for selection")
	}
	collection := newWorktreeCollection(entries, opts)
	root, err := gitx.Root("")
	if err != nil {
		root = ""
	}
	resolver := gitx.NewBranchStatusResolver(root)
	startWorktreeStatusLoader(collection, resolver)
	idx, err := fuzzyfinder.Find(&collection.slice, func(i int) string {
		return collection.itemString(i)
	},
		fuzzyfinder.WithPromptString("Select worktree to run command: "),
		fuzzyfinder.WithHotReloadLock(&collection.lock),
	)
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return errors.New("selection cancelled")
		}
		return err
	}
	entry, ok := collection.entryByIndex(idx)
	if !ok {
		return errors.New("selection cancelled")
	}
	return runInWorktree(entry.path, cmdArgs)
}

func runCmdForBranch(branch string, cmdArgs []string) error {
	p, err := gitx.FindWorktreeByBranch("", branch)
	if err != nil {
		return fmt.Errorf("no worktree found for branch: %s", branch)
	}
	return runInWorktree(p, cmdArgs)
}

func runInWorktree(path string, cmdArgs []string) error {
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
