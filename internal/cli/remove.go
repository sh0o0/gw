package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"

	"github.com/sh0o0/gw/internal/gitx"
	"github.com/spf13/cobra"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
)

func newRmCmd() *cobra.Command {
	var force bool
	var opts fuzzyDisplayOptions
	var merged bool
	var hookBackground bool
	var hookForeground bool
	var background bool
	var pathArg string
	cmd := &cobra.Command{
		Use:   "rm [--force] [branch ...]",
		Short: "Remove worktree(s) by fuzzy select or by branch names",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			effectiveHookBg := hookBackground || (cfg.HooksBackground && !hookForeground)

			removeOpts := removeOptions{force: force, hookBackground: effectiveHookBg, background: background}
			if pathArg != "" {
				return removeWorktreeForeground(pathArg, removeOpts)
			}
			if merged {
				if len(args) > 0 {
					return errors.New("--merged cannot be combined with branch arguments")
				}
				return removeMergedInteractive(removeOpts, opts)
			}
			if len(args) > 0 {
				success := 0
				var failed []string
				for _, b := range args {
					if err := removeWorktreeByBranch(b, removeOpts); err != nil {
						failed = append(failed, b)
						out.Error("Failed to remove worktree for branch: %s", b)
					} else {
						success++
						out.Success("Removed worktree for branch: %s", b)
					}
				}
				out.Summary(success, len(failed), "worktree")
				if len(failed) > 0 {
					return fmt.Errorf("failed branches: %s", strings.Join(failed, ", "))
				}
				return nil
			}
			return removeInteractive(removeOpts, opts)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "force remove")
	cmd.Flags().BoolVar(&opts.showPath, "show-path", false, "display worktree path in fuzzy finder")
	cmd.Flags().BoolVar(&merged, "merged", false, "remove all merged branches")
	cmd.Flags().BoolVar(&hookBackground, "hook-bg", false, "Run post-remove hook in background")
	cmd.Flags().BoolVar(&hookForeground, "hook-fg", false, "Run post-remove hook in foreground (override config)")
	cmd.Flags().BoolVar(&background, "bg", false, "Run removal in background")
	cmd.Flags().StringVar(&pathArg, "path", "", "Remove worktree by path (internal use)")
	cmd.Flags().MarkHidden("path")
	cmd.MarkFlagsMutuallyExclusive("hook-bg", "hook-fg")
	return cmd
}

type removeOptions struct {
	force          bool
	hookBackground bool
	background     bool
}

func removeWorktreeAtPath(path string, opts removeOptions) error {
	if opts.background {
		return removeWorktreeInBackground(path, opts)
	}
	return removeWorktreeForeground(path, opts)
}

func removeWorktreeInBackground(path string, opts removeOptions) error {
	args := []string{"rm", "--path", path}
	if opts.force {
		args = append(args, "--force")
	}
	if opts.hookBackground {
		args = append(args, "--hook-bg")
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	logFile, err := os.CreateTemp("", "gw-rm-*.log")
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	logPath := logFile.Name()

	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start background removal: %w", err)
	}
	out.Working("Started background removal for: %s %s", out.Highlight(path), out.Dim(fmt.Sprintf("(PID: %d, log: %s)", cmd.Process.Pid, logPath)))
	return nil
}

func removeWorktreeForeground(path string, opts removeOptions) error {
	br, _ := gitx.BranchAt(path)
	out.Trash("Removing worktree: %s", out.Highlight(path))
	if opts.force {
		if _, err := gitx.Cmd("", "worktree", "remove", "--force", path); err != nil {
			return err
		}
	} else {
		if _, err := gitx.Cmd("", "worktree", "remove", path); err != nil {
			return err
		}
	}
	if br != "" && br != "HEAD" {
		out.Branch("Deleting branch: %s", out.Highlight(br))
		if _, err := gitx.Cmd("", "branch", "-D", br); err != nil {
			out.Warn("Failed to delete branch: %s", br)
		} else {
			out.Success("Deleted branch: %s", br)
		}
	}
	runPostRemove(br, path, opts.hookBackground)
	return nil
}

func removeInteractive(rmOpts removeOptions, opts fuzzyDisplayOptions) error {
	wts, err := gitx.ListWorktrees("")
	if err != nil {
		return err
	}
	current, _ := gitx.CurrentWorktreePath("")
	primaryPath, _ := primaryWorktreePath()
	entries := buildWorktreeEntries(wts, func(wt gitx.Worktree) bool {
		return wt.Path == current || samePath(wt.Path, primaryPath)
	}, primaryPath)
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
	idxs, err := fuzzyfinder.FindMulti(&collection.slice, func(i int) string {
		return collection.itemString(i)
	},
		fuzzyfinder.WithPromptString("Select worktree(s) to remove (TAB to mark, ENTER to confirm):"),
		fuzzyfinder.WithHotReloadLock(&collection.lock),
	)
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return errors.New("selection cancelled")
		}
		return err
	}
	if len(idxs) == 0 {
		return errors.New("selection cancelled")
	}
	selectedIdx := make([]int, 0, len(idxs))
	seen := make(map[int]struct{}, len(idxs))
	for _, idx := range idxs {
		if entryIdx, ok := collection.baseIndex(idx); ok {
			if _, dup := seen[entryIdx]; dup {
				continue
			}
			seen[entryIdx] = struct{}{}
			selectedIdx = append(selectedIdx, entryIdx)
		}
	}
	if len(selectedIdx) == 0 {
		return errors.New("selection cancelled")
	}
	sort.Ints(selectedIdx)
	success := 0
	var failed []string
	for _, idx := range selectedIdx {
		e := collection.base[idx]
		path := e.path
		if err := removeWorktreeAtPath(path, rmOpts); err != nil {
			failed = append(failed, path)
			out.Error("Failed to remove worktree: %s\n  %v", path, err)
			continue
		}
		success++
		out.Success("Removed worktree: %s", path)
	}
	out.Summary(success, len(failed), "worktree")
	if len(failed) > 0 {
		return fmt.Errorf("failed worktrees: %s", strings.Join(failed, ", "))
	}
	return nil
}

func removeWorktreeByBranch(branch string, opts removeOptions) error {
	p, err := gitx.FindWorktreeByBranch("", branch)
	if err != nil {
		return fmt.Errorf("no worktree found for branch: %s", branch)
	}
	current, _ := gitx.CurrentWorktreePath("")
	if current == p {
		return fmt.Errorf("cannot remove current worktree for branch: %s", branch)
	}
	return removeWorktreeAtPath(p, opts)
}

var errSkipRemoval = errors.New("skip removal")

// removeMergedBranches is kept for non-interactive deletion of merged local branches.
// Currently unused from the CLI but retained for potential scripting.
func removeMergedBranches(opts removeOptions) error {
	branches, err := gitx.MergedBranches("")
	if err != nil {
		return err
	}
	primaryPath, _ := primaryWorktreePath()
	primaryBranch := ""
	if primaryPath != "" {
		if b, err := gitx.BranchAt(primaryPath); err == nil {
			primaryBranch = b
		}
	}
	currentPath, _ := gitx.CurrentWorktreePath("")
	currentBranch := ""
	if currentPath != "" {
		if b, err := gitx.BranchAt(currentPath); err == nil {
			currentBranch = b
		}
	}
	seen := make(map[string]struct{}, len(branches))
	success := 0
	var failed []string
	for _, raw := range branches {
		branch := strings.TrimSpace(raw)
		if branch == "" {
			continue
		}
		if _, dup := seen[branch]; dup {
			continue
		}
		seen[branch] = struct{}{}
		if branch == primaryBranch || branch == currentBranch {
			continue
		}
		if err := removeMergedBranch(branch, opts, primaryPath, currentPath); err != nil {
			if errors.Is(err, errSkipRemoval) {
				continue
			}
			failed = append(failed, branch)
			out.Error("Failed to remove branch: %s\n  %v", branch, err)
			continue
		}
		success++
		out.Success("Removed branch: %s", branch)
	}
	out.Summary(success, len(failed), "branch")
	if len(failed) > 0 {
		return fmt.Errorf("failed branches: %s", strings.Join(failed, ", "))
	}
	return nil
}

func removeMergedBranch(branch string, opts removeOptions, primaryPath, currentPath string) error {
	if p, err := gitx.FindWorktreeByBranch("", branch); err == nil {
		if samePath(p, primaryPath) {
			return errSkipRemoval
		}
		if samePath(p, currentPath) {
			return errSkipRemoval
		}
		return removeWorktreeAtPath(p, opts)
	}
	flag := "-d"
	if opts.force {
		flag = "-D"
	}
	if _, err := gitx.Cmd("", "branch", flag, branch); err != nil {
		return err
	}
	return nil
}

// removeMergedInteractive opens a multi-select with all worktrees and defaults to removing
// those with status MERGED when the user doesn't explicitly select any.
func removeMergedInteractive(rmOpts removeOptions, opts fuzzyDisplayOptions) error {
	wts, err := gitx.ListWorktrees("")
	if err != nil {
		return err
	}
	current, _ := gitx.CurrentWorktreePath("")
	primaryPath, _ := primaryWorktreePath()

	// Determine merged branches via PR status (gh); filter only MERGED.
	root, err := gitx.Root("")
	if err != nil {
		root = ""
	}
	resolver := gitx.NewBranchStatusResolver(root)

	// Build entries only for merged PR worktrees (exclude current and primary).
	mergedEntries := make([]*worktreeEntry, 0, len(wts))
	for _, wt := range wts {
		if wt.Path == current {
			continue
		}
		if samePath(wt.Path, primaryPath) {
			continue
		}
		if wt.Branch == "" || wt.Branch == "HEAD" {
			continue
		}
		st, err := resolver.Status(wt.Path, wt.Branch)
		if err != nil || st != gitx.BranchStatusMerged {
			continue
		}
		entry := &worktreeEntry{
			branch:    wt.Branch,
			rawBranch: wt.Branch,
			path:      wt.Path,
			isPrimary: false,
		}
		entry.status.Store(st.Display())
		mergedEntries = append(mergedEntries, entry)
	}
	if len(mergedEntries) == 0 {
		return errors.New("no merged worktrees to remove")
	}

	collection := newWorktreeCollection(mergedEntries, opts)

	idxs, err := fuzzyfinder.FindMulti(&collection.slice, func(i int) string {
		return collection.itemString(i)
	},
		fuzzyfinder.WithPromptString("Select MERGED worktree(s) to EXCLUDE (TAB to exclude, ENTER to remove all):"),
		fuzzyfinder.WithHotReloadLock(&collection.lock),
	)
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return errors.New("selection cancelled")
		}
		return err
	}

	// Map fuzzy indices to base indices (selected = excluded), de-dup.
	excludedIdx := make(map[int]struct{}, len(idxs))
	for _, idx := range idxs {
		if entryIdx, ok := collection.baseIndex(idx); ok {
			excludedIdx[entryIdx] = struct{}{}
		}
	}

	// Build list of indices to delete = all minus excluded.
	toDelete := make([]int, 0, len(collection.base))
	for i := range collection.base {
		if _, excluded := excludedIdx[i]; !excluded {
			toDelete = append(toDelete, i)
		}
	}
	if len(toDelete) == 0 {
		return errors.New("no worktrees selected for removal")
	}

	sort.Ints(toDelete)
	success := 0
	var failed []string
	for _, idx := range toDelete {
		e := collection.base[idx]
		path := e.path
		if err := removeWorktreeAtPath(path, rmOpts); err != nil {
			failed = append(failed, path)
			out.Error("Failed to remove worktree: %s\n  %v", path, err)
			continue
		}
		success++
		out.Success("Removed worktree: %s", path)
	}
	out.Summary(success, len(failed), "worktree")
	if len(failed) > 0 {
		return fmt.Errorf("failed worktrees: %s", strings.Join(failed, ", "))
	}
	return nil
}
