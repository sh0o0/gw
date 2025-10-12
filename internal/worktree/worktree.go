package worktree

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/sh0o0/gw/internal/fsutil"
	"github.com/sh0o0/gw/internal/gitx"
)

func SymlinkPatterns() []string {
	return []string{
		"**/.vscode/*",
		"**/.claude/*",
		"**/.env*",
		"**/.github/prompts/*.local.prompt.md",
		"**/.ignored/**",
		"**/.serena/**",
		"**/CLAUDE.local.md",
		"**/AGENTS.local.md",
	}
}

func ExcludePatterns() []string { return []string{"**/node_modules/*"} }

func shouldExclude(path string) bool {
	for _, p := range ExcludePatterns() {
		if ok, _ := doublestar.PathMatch(p, path); ok {
			return true
		}
	}
	return false
}

func matchAnyPattern(path string, patterns []string) bool {
	for _, p := range patterns {
		if ok, _ := doublestar.PathMatch(p, path); ok {
			return true
		}
	}
	return false
}

func ParseRemoteURL(cwd string) (domain, org, repo string, has bool, err error) {
	out, e := gitx.Cmd(cwd, "remote", "get-url", "origin")
	if e != nil || strings.TrimSpace(out) == "" {
		return "", "", "", false, nil
	}
	u := strings.TrimSpace(out)
	// git@domain:org/repo.git
	if strings.HasPrefix(u, "git@") {
		at := strings.IndexByte(u, '@')
		colon := strings.IndexByte(u, ':')
		if at > 0 && colon > at {
			domain = u[at+1 : colon]
			rest := u[colon+1:]
			parts := strings.Split(rest, "/")
			if len(parts) >= 2 {
				org = parts[0]
				repo = strings.TrimSuffix(parts[1], ".git")
				return domain, org, repo, true, nil
			}
		}
	}
	// http(s)://domain/org/repo.git or git://domain/org/repo.git
	for _, pref := range []string{"https://", "http://", "git://"} {
		if strings.HasPrefix(u, pref) {
			rest := strings.TrimPrefix(u, pref)
			parts := strings.Split(rest, "/")
			if len(parts) >= 3 {
				domain = parts[0]
				org = parts[1]
				repo = strings.TrimSuffix(parts[2], ".git")
				return domain, org, repo, true, nil
			}
		}
	}
	return "", "", "", false, fmt.Errorf("unsupported remote: %s", u)
}

func WorktreeBasePath(cwd string) (string, error) {
	root, err := gitx.Root(cwd)
	if err != nil {
		return "", err
	}
	if d, o, r, has, _ := ParseRemoteURL(cwd); has {
		return filepath.Join(os.Getenv("HOME"), ".worktrees", d, o, r), nil
	}
	rel := strings.TrimPrefix(root, os.Getenv("HOME")+"/")
	return filepath.Join(os.Getenv("HOME"), ".worktrees", "local", rel), nil
}

func ComputeWorktreePath(cwd, branch string) (string, error) {
	if branch == "" {
		return "", errors.New("branch required")
	}
	base, err := WorktreeBasePath(cwd)
	if err != nil {
		return "", err
	}
	if err := fsutil.EnsureDir(base); err != nil {
		return "", err
	}
	safe := strings.ReplaceAll(branch, "/", "-")
	p := filepath.Join(base, safe)
	if p == "/" || p == "" {
		return "", errors.New("invalid worktree path")
	}
	return p, nil
}

func GitIgnoredFiles(root string) ([]string, error) {
	out, err := gitx.Cmd(root, "ls-files", "--others", "-i", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	var res []string
	for _, ln := range strings.Split(strings.TrimSpace(out), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		res = append(res, ln)
	}
	return res, nil
}

func CreateSymlinksFromGitignored(root, target string) (int, error) {
	files, err := GitIgnoredFiles(root)
	if err != nil {
		return 0, err
	}
	pats := SymlinkPatterns()
	count := 0
	for _, f := range files {
		if shouldExclude(f) {
			continue
		}
		if !matchAnyPattern("/"+f, pats) { // fish version matches against leading slash
			continue
		}
		src := filepath.Join(root, f)
		dst := filepath.Join(target, f)
		if _, err := os.Lstat(src); err != nil {
			continue
		}

		// Resolve symlink chain to get the actual file
		actualSrc, err := filepath.EvalSymlinks(src)
		if err != nil {
			// If we can't resolve, use the original src
			actualSrc = src
		}

		if err := fsutil.CreateSymlink(actualSrc, dst); err != nil {
			return count, err
		}
		fmt.Fprintf(os.Stderr, "Created symlink: %s -> %s\n", dst, actualSrc)
		count++
	}
	return count, nil
}
