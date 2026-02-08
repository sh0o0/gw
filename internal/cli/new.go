package cli

import (
	"fmt"

	"github.com/sh0o0/gw/internal/gitx"
	"github.com/sh0o0/gw/internal/worktree"
	"github.com/spf13/cobra"
)

func newNewCmd() *cobra.Command {
	var fromRef string
	var fromCurrent bool
	var openEditor bool
	var noEditor bool
	var editorCmd string
	var hookBackground bool
	var hookForeground bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "new <branch>",
		Short: "Create new worktree with a new branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()

			effectiveOpenEditor := openEditor || (cfg.NewOpenEditor && !noEditor)
			effectiveHookBg := hookBackground || (cfg.HooksBackground && !hookForeground)
			branch := args[0]

			if path, err := gitx.FindWorktreeByBranch("", branch); err == nil && path != "" {
				return fmt.Errorf("worktree already exists for branch: %s", branch)
			}

			if exists, _ := gitx.BranchExists("", branch); exists {
				return fmt.Errorf("branch already exists: %s", branch)
			}

			baseRef := fromRef
			var symlinkSource string
			if fromCurrent {
				currentBranch, err := gitx.BranchAt(".")
				if err != nil {
					return fmt.Errorf("failed to get current branch: %w", err)
				}
				baseRef = currentBranch
				symlinkSource, _ = gitx.Root("")
			} else if fromRef != "" {
				if wtPath, err := gitx.FindWorktreeByBranch("", fromRef); err == nil {
					symlinkSource = wtPath
				}
			}
			if baseRef == "" {
				baseRef, _ = gitx.PrimaryBranch("")
			}

			p, err := worktree.ComputeWorktreePath("", branch)
			if err != nil {
				return err
			}
			if _, err := gitx.Cmd("", "worktree", "add", p, "-b", branch, baseRef); err != nil {
				return err
			}

			out.Branch("Created branch %s from %s", out.Highlight(branch), out.Highlight(baseRef))
			out.Folder("Worktree at %s", out.Highlight(p))

			if effectiveOpenEditor {
				editor := resolveEditor(editorCmd)
				if editor == "" {
					fmt.Fprintln(cmd.ErrOrStderr(), "Warning: no editor specified, skipping editor open")
				} else {
					if err := openEditorCmd(editor, p); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to open editor: %v\n", err)
					}
				}
			}

			if err := createSymlinks(p, PostCreateOptions{Verbose: verbose, SymlinkSource: symlinkSource}); err != nil {
				return err
			}

			runPostCreate(branch, p, effectiveHookBg)

			return navigateToWorktree(p)
		},
	}

	cmd.Flags().StringVar(&fromRef, "from", "", "Create from specific ref (branch, tag, or commit)")
	cmd.Flags().BoolVar(&fromCurrent, "from-current", false, "Create from current branch")
	cmd.Flags().BoolVarP(&openEditor, "editor", "e", false, "Open editor after creating worktree")
	cmd.Flags().BoolVar(&noEditor, "no-editor", false, "Do not open editor (override config)")
	cmd.Flags().StringVar(&editorCmd, "editor-cmd", "", "Editor command to use (default: $EDITOR)")
	cmd.Flags().BoolVar(&hookBackground, "hook-bg", false, "Run post-create hook in background")
	cmd.Flags().BoolVar(&hookForeground, "hook-fg", false, "Run post-create hook in foreground (override config)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show each symlink created")
	cmd.MarkFlagsMutuallyExclusive("from", "from-current")
	cmd.MarkFlagsMutuallyExclusive("editor", "no-editor")
	cmd.MarkFlagsMutuallyExclusive("hook-bg", "hook-fg")

	return cmd
}

func newAddCmd() *cobra.Command {
	var verbose bool
	var hookBackground bool
	var hookForeground bool
	var openEditor bool
	var noEditor bool
	var editorCmd string

	cmd := &cobra.Command{
		Use:   "add <branch>",
		Short: "Create new worktree for an existing branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			effectiveHookBg := hookBackground || (cfg.HooksBackground && !hookForeground)
			effectiveOpenEditor := openEditor || (cfg.AddOpenEditor && !noEditor)
			branch := args[0]

			gitx.Cmd("", "fetch", "origin", branch)

			p, err := worktree.ComputeWorktreePath("", branch)
			if err != nil {
				return err
			}
			if _, err := gitx.Cmd("", "worktree", "add", p, branch); err != nil {
				return err
			}

			out.Folder("Worktree at %s", out.Highlight(p))

			if effectiveOpenEditor {
				editor := resolveEditor(editorCmd)
				if editor == "" {
					fmt.Fprintln(cmd.ErrOrStderr(), "Warning: no editor specified, skipping editor open")
				} else {
					if err := openEditorCmd(editor, p); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to open editor: %v\n", err)
					}
				}
			}

			if err := createSymlinks(p, PostCreateOptions{Verbose: verbose}); err != nil {
				return err
			}

			runPostCreate(branch, p, effectiveHookBg)

			return navigateToWorktree(p)
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show each symlink created")
	cmd.Flags().BoolVar(&hookBackground, "hook-bg", false, "Run post-create hook in background")
	cmd.Flags().BoolVar(&hookForeground, "hook-fg", false, "Run post-create hook in foreground (override config)")
	cmd.Flags().BoolVarP(&openEditor, "editor", "e", false, "Open editor after creating worktree")
	cmd.Flags().BoolVar(&noEditor, "no-editor", false, "Do not open editor (override config)")
	cmd.Flags().StringVar(&editorCmd, "editor-cmd", "", "Editor command to use (default: $EDITOR)")
	cmd.MarkFlagsMutuallyExclusive("hook-bg", "hook-fg")
	cmd.MarkFlagsMutuallyExclusive("editor", "no-editor")
	return cmd
}
