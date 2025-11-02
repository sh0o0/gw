package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/sh0o0/gw/internal/fsutil"
	"github.com/sh0o0/gw/internal/fzfw"
	"github.com/sh0o0/gw/internal/gitx"
	"github.com/sh0o0/gw/internal/hooks"
	"github.com/sh0o0/gw/internal/worktree"
	"github.com/spf13/cobra"
)

func newLinkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "link <path>",
		Short: "Move file to base worktree and create symlink",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := fsutil.ResolveAbs(args[0])
			if err != nil {
				return err
			}
			if _, err := os.Lstat(p); err != nil {
				return fmt.Errorf("file not found: %s", p)
			}
			root, err := gitx.Root("")
			if err != nil {
				return err
			}
			primary, _ := primaryWorktreePath()
			if root == primary {
				return errors.New("you are in the primary worktree; nothing to link")
			}
			if !strings.HasPrefix(p, root+string(os.PathSeparator)) {
				return fmt.Errorf("path must be within current worktree: %s", root)
			}
			rel, _ := filepath.Rel(root, p)
			dst := filepath.Join(primary, rel)
			if _, err := os.Lstat(dst); err == nil {
				return fmt.Errorf("destination already exists: %s", dst)
			}
			if err := fsutil.EnsureDir(filepath.Dir(dst)); err != nil {
				return err
			}
			if err := os.Rename(p, dst); err != nil {
				return err
			}
			if err := os.Symlink(dst, p); err != nil {
				return err
			}
			fmt.Printf("Linked: %s -> %s\n", p, dst)
			return nil
		},
	}
}

func newUnlinkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unlink <path>",
		Short: "Replace symlink with a real file/dir by copying its target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := fsutil.ResolveAbs(args[0])
			if err != nil {
				return err
			}
			target, err := fsutil.MaterializeSymlink(p)
			if err != nil {
				return err
			}
			if target != "" {
				fmt.Printf("Unlinked: %s (copied from: %s)\n", p, target)
			} else {
				fmt.Printf("Unlinked: %s\n", p)
			}
			return nil
		},
	}
}

func newSwitchCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "switch [branch]",
		Short:   "Fuzzy search and cd to worktree or switch directly by branch",
		Aliases: []string{"sw"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return switchToBranch(args[0])
			}
			return switchInteractive(true)
		},
	}
}

func newCheckoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "checkout <branch>",
		Short:   "Switch to existing worktree or create new one for branch",
		Aliases: []string{"co"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prevRev, _ := gitx.Cmd("", "rev-parse", "--verify", "HEAD")
			prevRev = strings.TrimSpace(prevRev)
			prevBranch, _ := gitx.BranchAt(".")

			branch := args[0]
			path, err := gitx.FindWorktreeByBranch("", branch)
			if err == nil && path != "" {
				if err := switchToPath(path); err != nil {
					return err
				}
				// trigger post-checkout for existing worktree switch
				newRev, _ := gitx.Cmd(path, "rev-parse", "--verify", "HEAD")
				newRev = strings.TrimSpace(newRev)
				newBranch, _ := gitx.BranchAt(path)
				runPostCheckoutWithCWD(prevRev, newRev, prevBranch, newBranch, path)
				return nil
			}
			// create new worktree
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

func newRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <branch>",
		Short: "Create new worktree for specified branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			branch := args[0]
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

func newMoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "move <old-branch> <new-branch>",
		Short: "Rename a branch and relocate its worktree directory",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return moveWorktree(args[0], args[1])
		},
	}
}

func newRemoveCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     "remove [--force] [branch ...]",
		Short:   "Remove worktree(s) by fuzzy select or by branch names",
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				// remove multiple by branch
				success := 0
				var failed []string
				for _, b := range args {
					if err := removeWorktreeByBranch(b, force); err != nil {
						failed = append(failed, b)
						fmt.Fprintf(os.Stderr, "✗ Failed to remove worktree for branch: %s\n\n", b)
					} else {
						success++
						fmt.Fprintf(os.Stderr, "✓ Successfully removed worktree for branch: %s\n\n", b)
					}
				}
				fmt.Fprintf(os.Stderr, "Summary:\n  Successfully removed: %d worktree(s)\n", success)
				if len(failed) > 0 {
					return fmt.Errorf("failed branches: %s", strings.Join(failed, ", "))
				}
				return nil
			}
			// interactive
			return removeInteractive(force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "force remove")
	return cmd
}

func moveWorktree(oldBranch, newBranch string) error {
	if oldBranch == "" || newBranch == "" {
		return errors.New("both branch names required")
	}
	if oldBranch == newBranch {
		return errors.New("branch names must differ")
	}
	oldPath, err := gitx.FindWorktreeByBranch("", oldBranch)
	if err != nil {
		return fmt.Errorf("no worktree found for branch: %s", oldBranch)
	}
	oldPath = filepath.Clean(oldPath)

	sameDir := func(p1, p2 string) bool {
		if p1 == p2 {
			return true
		}
		info1, err1 := os.Stat(p1)
		info2, err2 := os.Stat(p2)
		if err1 == nil && err2 == nil && os.SameFile(info1, info2) {
			return true
		}
		r1, err1 := filepath.EvalSymlinks(p1)
		r2, err2 := filepath.EvalSymlinks(p2)
		return err1 == nil && err2 == nil && r1 == r2
	}

	if primary, err := primaryWorktreePath(); err == nil {
		if sameDir(primary, oldPath) {
			return errors.New("cannot move primary worktree")
		}
	}

	resolvedBranch, err := gitx.BranchAt(oldPath)
	if err != nil {
		return err
	}
	if resolvedBranch == "" || resolvedBranch == "HEAD" {
		return errors.New("cannot move detached worktree")
	}
	if resolvedBranch != oldBranch {
		return fmt.Errorf("worktree branch mismatch: expected %s, got %s", oldBranch, resolvedBranch)
	}

	destPath, err := worktree.ComputeWorktreePath(oldPath, newBranch)
	if err != nil {
		return err
	}
	destPath = filepath.Clean(destPath)
	samePath := sameDir(destPath, oldPath)

	if !samePath {
		if err := fsutil.EnsureDir(filepath.Dir(destPath)); err != nil {
			return err
		}
		if _, statErr := os.Stat(destPath); statErr == nil {
			return fmt.Errorf("destination already exists: %s", destPath)
		} else if !errors.Is(statErr, fs.ErrNotExist) {
			return statErr
		}
	}

	caller, err := callerCWD()
	if err != nil {
		return err
	}
	caller = filepath.Clean(caller)
	inOld := caller == oldPath || strings.HasPrefix(caller, oldPath+string(os.PathSeparator))
	relWithin := "."
	if inOld {
		if rel, relErr := filepath.Rel(oldPath, caller); relErr == nil {
			relWithin = rel
		}
	}

	if _, err := gitx.Cmd(oldPath, "branch", "-m", newBranch); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Renamed branch: %s -> %s\n", oldBranch, newBranch)

	if !samePath {
		if _, err := gitx.Cmd("", "worktree", "move", oldPath, destPath); err != nil {
			if _, revertErr := gitx.Cmd(oldPath, "branch", "-m", oldBranch); revertErr == nil {
				fmt.Fprintln(os.Stderr, "Reverted branch rename because worktree move failed")
			}
			return err
		}
		fmt.Fprintf(os.Stderr, "Moved worktree: %s -> %s\n", oldPath, destPath)
	} else {
		destPath = oldPath
	}

	printPath := destPath
	if inOld && relWithin != "." {
		candidate := filepath.Join(destPath, relWithin)
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			printPath = candidate
		}
	}
	fmt.Println(printPath)
	return nil
}

// helpers
func primaryWorktreePath() (string, error) {
	cg, err := gitx.CommonGitDir("")
	if err != nil {
		return "", err
	}
	p := filepath.Dir(cg)
	if fi, err := os.Stat(p); err == nil && fi.IsDir() {
		return p, nil
	}
	return "", errors.New("primary worktree not found")
}

func callerCWD() (string, error) {
	if v := os.Getenv("GW_CALLER_CWD"); v != "" {
		if filepath.IsAbs(v) {
			return filepath.Clean(v), nil
		}
		if abs, err := filepath.Abs(v); err == nil {
			return abs, nil
		}
		return "", fmt.Errorf("invalid GW_CALLER_CWD: %s", v)
	}
	return os.Getwd()
}

func relativePathFromGitRoot() (string, error) {
	root, err := gitx.Root("")
	if err != nil {
		return "", err
	}
	cwd, err := callerCWD()
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(cwd, root+string(os.PathSeparator)) {
		rel, _ := filepath.Rel(root, cwd)
		if rel == "." {
			return ".", nil
		}
		return rel, nil
	}
	if cwd == root {
		return ".", nil
	}
	return ".", nil
}

func navigateToRelativePath(worktreePath, rel string) error {
	target := worktreePath
	if rel != "." {
		candidate := filepath.Join(worktreePath, rel)
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			target = candidate
		}
	}
	// print path to stdout so shell function or user can cd; in Go we cannot change parent shell cwd
	fmt.Println(target)
	return nil
}

func postCreateWorktree(p string) error {
	root, err := gitx.Root("")
	if err != nil {
		return err
	}
	if _, err := worktree.CreateSymlinksFromGitignored(root, p); err != nil {
		return err
	}
	rel, _ := relativePathFromGitRoot()
	return navigateToRelativePath(p, rel)
}

func switchToPath(p string) error {
	current, _ := gitx.CurrentWorktreePath("")
	curBranch, _ := gitx.BranchAt(current)
	tgtBranch, _ := gitx.BranchAt(p)
	rel, _ := relativePathFromGitRoot()
	if err := navigateToRelativePath(p, rel); err != nil {
		return err
	}
	if curBranch != "" && tgtBranch != "" {
		fmt.Fprintf(os.Stderr, "Switched from [%s] to [%s]\n", curBranch, tgtBranch)
	} else {
		fmt.Fprintf(os.Stderr, "Switched to worktree: %s\n", p)
	}
	return nil
}

func switchInteractive(excludeCurrent bool) error {
	wts, err := gitx.ListWorktrees("")
	if err != nil {
		return err
	}
	current, _ := gitx.CurrentWorktreePath("")
	root, err := gitx.Root("")
	if err != nil {
		root = ""
	}
	resolver := gitx.NewBranchStatusResolver(root)
	display, paths, err := prepareWorktreeSelection(wts, func(wt gitx.Worktree) bool {
		return excludeCurrent && wt.Path == current
	}, resolver)
	if err != nil {
		return err
	}
	if len(display) == 0 {
		return errors.New("no worktrees available for selection")
	}
	sel, err := fzfw.Select("Select worktree: ", display)
	if err != nil {
		return err
	}
	for i, d := range display {
		if d == sel {
			return switchToPath(paths[i])
		}
	}
	return errors.New("selection cancelled")
}

func switchToBranch(branch string) error {
	p, err := gitx.FindWorktreeByBranch("", branch)
	if err != nil {
		return fmt.Errorf("no worktree found for branch: %s", branch)
	}
	return switchToPath(p)
}

type worktreeOption struct {
	branch    string
	path      string
	rawBranch string
	status    gitx.BranchStatus
}

func prepareWorktreeSelection(wts []gitx.Worktree, skip func(gitx.Worktree) bool, resolver *gitx.BranchStatusResolver) ([]string, []string, error) {
	entries := make([]*worktreeOption, 0, len(wts))
	for _, wt := range wts {
		if skip != nil && skip(wt) {
			continue
		}
		branchLabel := wt.Branch
		if branchLabel == "" || branchLabel == "HEAD" {
			branchLabel = "(detached)"
		}
		entries = append(entries, &worktreeOption{
			branch:    branchLabel,
			path:      wt.Path,
			rawBranch: wt.Branch,
		})
	}
	if resolver != nil {
		if err := populateWorktreeStatuses(entries, resolver); err != nil {
			return nil, nil, err
		}
	}
	maxStatusLen := 0
	for _, entry := range entries {
		if l := len(entry.status.Display()); l > maxStatusLen {
			maxStatusLen = l
		}
	}
	display := make([]string, len(entries))
	paths := make([]string, len(entries))
	for i, entry := range entries {
		if maxStatusLen > 0 {
			display[i] = fmt.Sprintf("[%s]  %-*s %s", entry.branch, maxStatusLen, entry.status.Display(), entry.path)
		} else {
			display[i] = fmt.Sprintf("[%s]  %s", entry.branch, entry.path)
		}
		paths[i] = entry.path
	}
	return display, paths, nil
}

func populateWorktreeStatuses(entries []*worktreeOption, resolver *gitx.BranchStatusResolver) error {
	targets := make([]*worktreeOption, 0, len(entries))
	for _, entry := range entries {
		if entry.rawBranch == "" || entry.rawBranch == "HEAD" {
			continue
		}
		targets = append(targets, entry)
	}
	if len(targets) == 0 {
		return nil
	}
	limit := runtime.NumCPU()
	if limit > len(targets) {
		limit = len(targets)
	}
	if limit < 1 {
		limit = 1
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error
	for _, entry := range targets {
		entry := entry
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			status, err := resolver.Status(entry.path, entry.rawBranch)
			if err != nil {
				once.Do(func() { firstErr = err })
				return
			}
			entry.status = status
		}()
	}
	wg.Wait()
	return firstErr
}

func removeWorktreeAtPath(path string, force bool) error {
	br, _ := gitx.BranchAt(path)
	fmt.Fprintf(os.Stderr, "Removing worktree: %s\n", path)
	if force {
		if _, err := gitx.Cmd("", "worktree", "remove", "--force", path); err != nil {
			return err
		}
	} else {
		if _, err := gitx.Cmd("", "worktree", "remove", path); err != nil {
			return err
		}
	}
	if br != "" && br != "HEAD" {
		fmt.Fprintf(os.Stderr, "Deleting branch: %s\n", br)
		if _, err := gitx.Cmd("", "branch", "-D", br); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to delete branch: %s\n", br)
		} else {
			fmt.Fprintf(os.Stderr, "Successfully deleted branch: %s\n", br)
		}
	}
	return nil
}

func removeInteractive(force bool) error {
	// choose non-current
	wts, err := gitx.ListWorktrees("")
	if err != nil {
		return err
	}
	current, _ := gitx.CurrentWorktreePath("")
	root, err := gitx.Root("")
	if err != nil {
		root = ""
	}
	resolver := gitx.NewBranchStatusResolver(root)
	display, paths, err := prepareWorktreeSelection(wts, func(wt gitx.Worktree) bool {
		return wt.Path == current
	}, resolver)
	if err != nil {
		return err
	}
	if len(display) == 0 {
		return errors.New("no worktrees available for selection")
	}
	sels, err := fzfw.SelectMultiple("Select worktree(s) to remove (TAB to mark, ENTER to confirm):", display)
	if err != nil {
		return err
	}
	if len(sels) == 0 {
		return errors.New("selection cancelled")
	}
	indexMap := make(map[string]int, len(display))
	for i, d := range display {
		indexMap[d] = i
	}
	success := 0
	var failed []string
	for _, sel := range sels {
		idx := indexMap[sel]
		path := paths[idx]
		if err := removeWorktreeAtPath(path, force); err != nil {
			failed = append(failed, path)
			fmt.Fprintf(os.Stderr, "✗ Failed to remove worktree: %s\n  %v\n\n", path, err)
			continue
		}
		success++
		fmt.Fprintf(os.Stderr, "✓ Successfully removed worktree: %s\n\n", path)
	}
	fmt.Fprintf(os.Stderr, "Summary:\n  Successfully removed: %d worktree(s)\n", success)
	if len(failed) > 0 {
		return fmt.Errorf("failed worktrees: %s", strings.Join(failed, ", "))
	}
	return nil
}

func removeWorktreeByBranch(branch string, force bool) error {
	p, err := gitx.FindWorktreeByBranch("", branch)
	if err != nil {
		return fmt.Errorf("no worktree found for branch: %s", branch)
	}
	// Cannot remove current worktree
	current, _ := gitx.CurrentWorktreePath("")
	if current == p {
		return fmt.Errorf("cannot remove current worktree for branch: %s", branch)
	}
	return removeWorktreeAtPath(p, force)
}

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

func runPostCheckout(prevRev, newRev, prevBranch, newBranch string) {
	// env
	gitRoot, _ := gitx.Root("")
	primary, _ := primaryWorktreePath()
	d := hooks.HooksDir(primary, gitRoot)
	env := map[string]string{
		"GW_HOOK_NAME":   "post-checkout",
		"GW_PREV_BRANCH": prevBranch,
		"GW_NEW_BRANCH":  newBranch,
	}
	if ran, err := hooks.RunHook(d, "post-checkout", env, prevRev, newRev, "1"); ran {
		if err != nil {
			fmt.Fprintln(os.Stderr, "Warning: post-checkout hook completed with errors")
		} else {
			fmt.Fprintln(os.Stderr, "post-checkout hook executed")
		}
	}
}

// runPostCheckoutWithCWD is like runPostCheckout but ensures hooks run with working directory
// set to the target worktree path, so package managers (e.g., pnpm) operate on that tree.
func runPostCheckoutWithCWD(prevRev, newRev, prevBranch, newBranch, worktreePath string) {
	gitRoot, _ := gitx.Root("")
	primary, _ := primaryWorktreePath()
	d := hooks.HooksDir(primary, gitRoot)
	env := map[string]string{
		"GW_HOOK_NAME":   "post-checkout",
		"GW_PREV_BRANCH": prevBranch,
		"GW_NEW_BRANCH":  newBranch,
		"GW_HOOK_CWD":    worktreePath,
	}
	if ran, err := hooks.RunHook(d, "post-checkout", env, prevRev, newRev, "1"); ran {
		if err != nil {
			fmt.Fprintln(os.Stderr, "Warning: post-checkout hook completed with errors")
		} else {
			fmt.Fprintln(os.Stderr, "post-checkout hook executed")
		}
	}
}
