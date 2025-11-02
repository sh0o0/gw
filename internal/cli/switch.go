package cli

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sh0o0/gw/internal/gitx"
	"github.com/spf13/cobra"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
)

func newSwitchCmd() *cobra.Command {
	var opts fuzzyDisplayOptions
	cmd := &cobra.Command{
		Use:     "switch [branch]",
		Short:   "Fuzzy search and cd to worktree or switch directly by branch",
		Aliases: []string{"sw"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return switchToBranch(args[0])
			}
			return switchInteractive(true, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.showPath, "show-path", false, "display worktree path in fuzzy finder")
	return cmd
}

func switchInteractive(excludeCurrent bool, opts fuzzyDisplayOptions) error {
	wts, err := gitx.ListWorktrees("")
	if err != nil {
		return err
	}
	current, _ := gitx.CurrentWorktreePath("")
	entries := buildWorktreeEntries(wts, func(wt gitx.Worktree) bool {
		return excludeCurrent && wt.Path == current
	})
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
		fuzzyfinder.WithPromptString("Select worktree: "),
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
	return switchToPath(entry.path)
}

func switchToBranch(branch string) error {
	p, err := gitx.FindWorktreeByBranch("", branch)
	if err != nil {
		return fmt.Errorf("no worktree found for branch: %s", branch)
	}
	return switchToPath(p)
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

const (
	statusColumnWidth    = len("IN PROGRESS")
	loadingStatusDisplay = "LOADING"
)

type worktreeEntry struct {
	branch    string
	rawBranch string
	path      string
	status    atomic.Value
}

func buildWorktreeEntries(wts []gitx.Worktree, skip func(gitx.Worktree) bool) []*worktreeEntry {
	entries := make([]*worktreeEntry, 0, len(wts))
	for _, wt := range wts {
		if skip != nil && skip(wt) {
			continue
		}
		branchLabel := wt.Branch
		if branchLabel == "" || branchLabel == "HEAD" {
			branchLabel = "(detached)"
		}
		initial := ""
		if wt.Branch != "" && wt.Branch != "HEAD" {
			initial = loadingStatusDisplay
		}
		entry := &worktreeEntry{
			branch:    branchLabel,
			rawBranch: wt.Branch,
			path:      wt.Path,
		}
		entry.status.Store(initial)
		entries = append(entries, entry)
	}
	return entries
}

func (e *worktreeEntry) display(opts fuzzyDisplayOptions) string {
	status := ""
	if v := e.status.Load(); v != nil {
		status = v.(string)
	}
	label := formatStatusLabel(status)

	var b strings.Builder
	b.WriteString("[")
	b.WriteString(e.branch)
	b.WriteString("]")

	if label != "" || opts.showPath {
		b.WriteString("  ")
	}

	if label != "" {
		b.WriteString(label)
	}

	if opts.showPath {
		if label != "" {
			b.WriteByte(' ')
		}
		b.WriteString(e.path)
	}

	return b.String()
}

func formatStatusLabel(status string) string {
	if status == "" {
		return ""
	}
	return fmt.Sprintf("%-*s", statusColumnWidth, status)
}

type worktreeCollection struct {
	base    []*worktreeEntry
	slice   []*worktreeEntry
	lock    sync.Mutex
	toggled bool
	options fuzzyDisplayOptions
}

func newWorktreeCollection(entries []*worktreeEntry, opts fuzzyDisplayOptions) *worktreeCollection {
	base := append([]*worktreeEntry(nil), entries...)
	slice := append([]*worktreeEntry(nil), entries...)
	return &worktreeCollection{
		base:    base,
		slice:   slice,
		options: opts,
	}
}

func (c *worktreeCollection) itemString(i int) string {
	if i < len(c.base) {
		return c.base[i].display(c.options)
	}
	return ""
}

func (c *worktreeCollection) entryByIndex(i int) (*worktreeEntry, bool) {
	if i < len(c.base) {
		return c.base[i], true
	}
	return nil, false
}

func (c *worktreeCollection) baseIndex(i int) (int, bool) {
	if i < len(c.base) {
		return i, true
	}
	return -1, false
}

func (c *worktreeCollection) triggerReload() {
	c.lock.Lock()
	if c.toggled {
		c.lock.Unlock()
		return
	}
	c.slice = append(c.slice, nil)
	c.toggled = true
	c.lock.Unlock()

	go func() {
		time.Sleep(60 * time.Millisecond)
		c.lock.Lock()
		c.slice = c.slice[:len(c.base)]
		c.toggled = false
		c.lock.Unlock()
	}()
}

func (c *worktreeCollection) finalize() {
	c.lock.Lock()
	c.slice = c.slice[:len(c.base)]
	c.toggled = false
	c.lock.Unlock()
}

func startWorktreeStatusLoader(collection *worktreeCollection, resolver *gitx.BranchStatusResolver) {
	if resolver == nil {
		return
	}
	go func() {
		limit := runtime.NumCPU()
		if limit < 1 {
			limit = 1
		}
		sem := make(chan struct{}, limit)
		var wg sync.WaitGroup
		for _, entry := range collection.base {
			if entry.rawBranch == "" || entry.rawBranch == "HEAD" {
				continue
			}
			entry := entry
			wg.Add(1)
			go func() {
				sem <- struct{}{}
				defer func() {
					<-sem
					wg.Done()
				}()
				status, err := resolver.Status(entry.path, entry.rawBranch)
				if err != nil {
					return
				}
				newStatus := status.Display()
				current := ""
				if v := entry.status.Load(); v != nil {
					current, _ = v.(string)
				}
				if current != newStatus {
					entry.status.Store(newStatus)
					collection.triggerReload()
				}
			}()
		}
		wg.Wait()
		collection.finalize()
	}()
}
