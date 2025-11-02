package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sh0o0/gw/internal/gitx"
	"github.com/sh0o0/gw/internal/worktree"
)

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

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return filepath.Clean(a) == filepath.Clean(b)
}
