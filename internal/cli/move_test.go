package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sh0o0/gw/internal/worktree"
)

func TestMoveWorktree_shouldRenameBranchAndMoveDirectory_whenBranchChanges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := filepath.Join(home, "repo")
	initTestRepo(t, repo)

	oldBranch := "feature/old"
	oldPath, err := worktree.ComputeWorktreePath(repo, oldBranch)
	if err != nil {
		t.Fatalf("compute old path: %v", err)
	}
	runGit(t, repo, "worktree", "add", oldPath, "-b", oldBranch)

	nested := filepath.Join(oldPath, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	t.Setenv("GW_CALLER_CWD", nested)

	newBranch := "feature/new"
	if err := moveWorktree(oldBranch, newBranch); err != nil {
		t.Fatalf("moveWorktree: %v", err)
	}

	newPath, err := worktree.ComputeWorktreePath(repo, newBranch)
	if err != nil {
		t.Fatalf("compute new path: %v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected path missing: %v", err)
	}
	if _, err := os.Stat(oldPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old path should be removed, err=%v", err)
	}
	branchOut := strings.TrimSpace(runGitOutput(t, newPath, "branch", "--show-current"))
	if branchOut != newBranch {
		t.Fatalf("unexpected branch: want %s got %s", newBranch, branchOut)
	}
}

func TestMoveWorktree_shouldRenameBranchOnly_whenSanitizedPathMatches(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := filepath.Join(home, "repo")
	initTestRepo(t, repo)

	oldBranch := "feature/foo"
	oldPath, err := worktree.ComputeWorktreePath(repo, oldBranch)
	if err != nil {
		t.Fatalf("compute old path: %v", err)
	}
	runGit(t, repo, "worktree", "add", oldPath, "-b", oldBranch)
	t.Setenv("GW_CALLER_CWD", oldPath)

	newBranch := "feature-foo"
	if err := moveWorktree(oldBranch, newBranch); err != nil {
		t.Fatalf("moveWorktree: %v", err)
	}

	newPath, err := worktree.ComputeWorktreePath(repo, newBranch)
	if err != nil {
		t.Fatalf("compute new path: %v", err)
	}
	if newPath != oldPath {
		t.Fatalf("expected paths to match, old=%s new=%s", oldPath, newPath)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected path missing: %v", err)
	}
	branchOut := strings.TrimSpace(runGitOutput(t, newPath, "branch", "--show-current"))
	if branchOut != newBranch {
		t.Fatalf("unexpected branch: want %s got %s", newBranch, branchOut)
	}
}

func TestMoveWorktree_shouldReturnError_whenInputInvalid(t *testing.T) {
	tests := []struct {
		name      string
		oldBranch string
		newBranch string
		setup     func(t *testing.T)
	}{
		{
			name:      "shouldErrorWhenBranchesMatch",
			oldBranch: "feature/dup",
			newBranch: "feature/dup",
			setup: func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
				repo := filepath.Join(home, "repo")
				initTestRepo(t, repo)
				t.Setenv("GW_CALLER_CWD", repo)
			},
		},
		{
			name:      "shouldErrorWhenWorktreeMissing",
			oldBranch: "feature/missing",
			newBranch: "feature/new",
			setup: func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
				repo := filepath.Join(home, "repo")
				initTestRepo(t, repo)
				t.Setenv("GW_CALLER_CWD", repo)
			},
		},
		{
			name:      "shouldErrorWhenPrimaryWorktree",
			oldBranch: "main",
			newBranch: "main-renamed",
			setup: func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
				repo := filepath.Join(home, "repo")
				initTestRepo(t, repo)
				t.Setenv("GW_CALLER_CWD", repo)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(t)
			if err := moveWorktree(tc.oldBranch, tc.newBranch); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

func initTestRepo(t *testing.T, repo string) {
	t.Helper()
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGit(t, filepath.Dir(repo), "init", "--initial-branch=main", repo)
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "remote", "add", "origin", "git@github.com:example/repo.git")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("test"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "init")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", args, err, out)
	}
}

func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", args, err, out)
	}
	return string(out)
}
