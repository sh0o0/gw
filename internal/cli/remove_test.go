package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sh0o0/gw/internal/worktree"
)

func TestRemoveMergedBranches_shouldRemoveBranches_whenMergedBranchesExist(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := filepath.Join(home, "repo")
	initTestRepo(t, repo)
	t.Setenv("GW_CALLER_CWD", repo)

	withWorktree := "feature/with-worktree"
	withPath, err := worktree.ComputeWorktreePath(repo, withWorktree)
	if err != nil {
		t.Fatalf("compute worktree path: %v", err)
	}
	runGit(t, repo, "worktree", "add", withPath, "-b", withWorktree)

	featureFile := filepath.Join(withPath, "feature.txt")
	if err := os.WriteFile(featureFile, []byte("feature"), 0o644); err != nil {
		t.Fatalf("write feature file: %v", err)
	}
	runGit(t, withPath, "add", "feature.txt")
	runGit(t, withPath, "commit", "-m", "feat: add feature worktree branch")
	runGit(t, repo, "switch", "main")
	runGit(t, repo, "merge", "--no-ff", withWorktree)

	withoutWorktree := "feature/without-worktree"
	runGit(t, repo, "switch", "-c", withoutWorktree)
	withoutFile := filepath.Join(repo, "without_worktree.txt")
	if err := os.WriteFile(withoutFile, []byte("without"), 0o644); err != nil {
		t.Fatalf("write without file: %v", err)
	}
	runGit(t, repo, "add", "without_worktree.txt")
	runGit(t, repo, "commit", "-m", "feat: add without worktree branch")
	runGit(t, repo, "switch", "main")
	runGit(t, repo, "merge", "--no-ff", withoutWorktree)

	if err := removeMergedBranches(false); err != nil {
		t.Fatalf("removeMergedBranches: %v", err)
	}

	branches := runGitOutput(t, repo, "branch")
	if strings.Contains(branches, withWorktree) {
		t.Fatalf("expected merged branch with worktree removed")
	}
	if strings.Contains(branches, withoutWorktree) {
		t.Fatalf("expected merged branch without worktree removed")
	}
	if _, err := os.Stat(withPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected worktree directory removed, err=%v", err)
	}
}
