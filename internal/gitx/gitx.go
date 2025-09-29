package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Cmd runs a git command in cwd (empty means current) and returns stdout as string.
func Cmd(cwd string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir := effectiveCWD(cwd); dir != "" {
		cmd.Dir = dir
	}
	var out bytes.Buffer
	var errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		if errb.Len() > 0 {
			return "", fmt.Errorf("git %v failed: %w: %s", args, err, strings.TrimSpace(errb.String()))
		}
		return "", fmt.Errorf("git %v failed: %w", args, err)
	}
	return out.String(), nil
}

func effectiveCWD(cwd string) string {
	if cwd != "" {
		return cwd
	}
	if env := os.Getenv("GW_CALLER_CWD"); env != "" {
		if filepath.IsAbs(env) {
			return filepath.Clean(env)
		}
		if abs, err := filepath.Abs(env); err == nil {
			return abs
		}
	}
	return ""
}

func InRepo(cwd string) bool {
	_, err := Cmd(cwd, "rev-parse", "--git-dir")
	return err == nil
}

func Root(cwd string) (string, error) {
	if !InRepo(cwd) {
		return "", errors.New("not in git repository")
	}
	out, err := Cmd(cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func CommonGitDir(cwd string) (string, error) {
	out, err := Cmd(cwd, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	p := strings.TrimSpace(out)
	if !filepath.IsAbs(p) {
		if cwd == "" {
			var err2 error
			cwd, err2 = os.Getwd()
			if err2 != nil {
				return "", err2
			}
		}
		p = filepath.Clean(filepath.Join(cwd, p))
	}
	return p, nil
}

type Worktree struct {
	Path   string
	Branch string // empty => detached/unknown, "HEAD" for detached head
}

// ListWorktrees returns parsed worktrees from `git worktree list --porcelain`.
func ListWorktrees(cwd string) ([]Worktree, error) {
	out, err := Cmd(cwd, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\n")
	var wts []Worktree
	var cur Worktree
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		switch {
		case ln == "":
			if cur.Path != "" {
				wts = append(wts, cur)
			}
			cur = Worktree{}
		case strings.HasPrefix(ln, "worktree "):
			cur.Path = strings.TrimPrefix(ln, "worktree ")
		case strings.HasPrefix(ln, "branch "):
			b := strings.TrimPrefix(ln, "branch ")
			b = strings.TrimPrefix(b, "refs/heads/")
			cur.Branch = b
		case strings.HasPrefix(ln, "HEAD "):
			cur.Branch = "HEAD"
		}
	}
	if cur.Path != "" {
		wts = append(wts, cur)
	}
	return wts, nil
}

func CurrentWorktreePath(cwd string) (string, error) {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	wts, err := ListWorktrees(cwd)
	if err != nil {
		return "", err
	}
	best := ""
	for _, wt := range wts {
		p := wt.Path
		if strings.HasPrefix(cwd, p) {
			if len(p) > len(best) {
				best = p
			}
		}
	}
	if best == "" {
		return "", errors.New("could not determine current worktree")
	}
	return best, nil
}

func BranchAt(path string) (string, error) {
	out, err := Cmd(path, "branch", "--show-current")
	if err == nil {
		b := strings.TrimSpace(out)
		if b != "" {
			return b, nil
		}
	}
	// fallback via list
	wts, err := ListWorktrees(path)
	if err != nil {
		return "", err
	}
	for _, wt := range wts {
		if wt.Path == path {
			return wt.Branch, nil
		}
	}
	return "", errors.New("branch not found")
}

func FindWorktreeByBranch(cwd, branch string) (string, error) {
	wts, err := ListWorktrees(cwd)
	if err != nil {
		return "", err
	}
	for _, wt := range wts {
		if wt.Branch == branch {
			return wt.Path, nil
		}
	}
	return "", fmt.Errorf("no worktree for branch %s", branch)
}
