package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sh0o0/gw/internal/worktree"
)

func TestRunCmdForBranch_shouldRunCommand_whenBranchExists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := filepath.Join(home, "repo")
	initTestRepo(t, repo)
	t.Setenv("GW_CALLER_CWD", repo)

	branch := "feature/test-run"
	wtPath, err := worktree.ComputeWorktreePath(repo, branch)
	if err != nil {
		t.Fatalf("compute worktree path: %v", err)
	}
	runGit(t, repo, "worktree", "add", wtPath, "-b", branch)

	outputFile := filepath.Join(home, "output.txt")
	err = runCmdForBranch(branch, []string{"touch", outputFile})
	if err != nil {
		t.Fatalf("runCmdForBranch: %v", err)
	}

	if _, err := os.Stat(outputFile); err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
}

func TestRunCmdForBranch_shouldReturnError_whenBranchNotExists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := filepath.Join(home, "repo")
	initTestRepo(t, repo)
	t.Setenv("GW_CALLER_CWD", repo)

	err := runCmdForBranch("nonexistent-branch", []string{"echo", "test"})
	if err == nil {
		t.Fatal("expected error for nonexistent branch")
	}
}

func TestRunInWorktree_shouldExecuteCommandInSpecifiedDirectory(t *testing.T) {
	home := t.TempDir()
	workdir := filepath.Join(home, "workdir")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}

	testFile := filepath.Join(workdir, "test-file.txt")
	err := runInWorktree(workdir, []string{"touch", testFile})
	if err != nil {
		t.Fatalf("runInWorktree: %v", err)
	}

	if _, err := os.Stat(testFile); err != nil {
		t.Fatalf("expected test file to exist: %v", err)
	}
}

func TestRunInWorktree_shouldReturnError_whenCommandFails(t *testing.T) {
	home := t.TempDir()

	err := runInWorktree(home, []string{"false"})
	if err == nil {
		t.Fatal("expected error for failed command")
	}
}
