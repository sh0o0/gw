package gitx

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBranchStatusResolver_shouldReportStatus_whenConditionChanges(t *testing.T) {
	const branchName = "feature/status"
	cases := []struct {
		name  string
		setup func(t *testing.T) (repo, branchPath, branch string)
		want  BranchStatus
	}{
		{
			name: "shouldReturnNotStartedWhenBranchClean",
			setup: func(t *testing.T) (string, string, string) {
				repo, branchPath := initStatusTestRepo(t, branchName)
				return repo, branchPath, branchName
			},
			want: BranchStatusNotStarted,
		},
		{
			name: "shouldReturnInProgressWhenWorktreeDirty",
			setup: func(t *testing.T) (string, string, string) {
				repo, branchPath := initStatusTestRepo(t, branchName)
				if err := os.WriteFile(filepath.Join(branchPath, "draft.txt"), []byte("todo"), 0o644); err != nil {
					t.Fatalf("write draft: %v", err)
				}
				return repo, branchPath, branchName
			},
			want: BranchStatusInProgress,
		},
		{
			name: "shouldReturnInProgressWhenBranchAheadOfBase",
			setup: func(t *testing.T) (string, string, string) {
				repo, branchPath := initStatusTestRepo(t, branchName)
				path := filepath.Join(branchPath, "work.txt")
				if err := os.WriteFile(path, []byte("work"), 0o644); err != nil {
					t.Fatalf("write work: %v", err)
				}
				runGitTestHelper(t, branchPath, "add", "work.txt")
				runGitTestHelper(t, branchPath, "commit", "-m", "feat: add work")
				return repo, branchPath, branchName
			},
			want: BranchStatusInProgress,
		},
		{
			name: "shouldReturnNotStartedWhenBranchMergedWithoutPR",
			setup: func(t *testing.T) (string, string, string) {
				repo, branchPath := initStatusTestRepo(t, branchName)
				path := filepath.Join(branchPath, "merged.txt")
				if err := os.WriteFile(path, []byte("merged"), 0o644); err != nil {
					t.Fatalf("write merged: %v", err)
				}
				runGitTestHelper(t, branchPath, "add", "merged.txt")
				runGitTestHelper(t, branchPath, "commit", "-m", "feat: merge work")
				runGitTestHelper(t, repo, "merge", "--ff-only", branchName)
				return repo, branchPath, branchName
			},
			want: BranchStatusNotStarted,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, branchPath, branch := tc.setup(t)
			resolver := NewBranchStatusResolver(repo)
			got, err := resolver.Status(branchPath, branch)
			if err != nil {
				t.Fatalf("Status error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected status: want %q got %q", tc.want, got)
			}
		})
	}
}

func TestBranchStatusResolver_shouldReturnOpened_whenGhReportsOpenPR(t *testing.T) {
	const branchName = "feature/opened"
	repo, branchPath := initStatusTestRepo(t, branchName)
	path := filepath.Join(branchPath, "feature.txt")
	if err := os.WriteFile(path, []byte("feature"), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	runGitTestHelper(t, branchPath, "add", "feature.txt")
	runGitTestHelper(t, branchPath, "commit", "-m", "feat: open pr")

	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	ghPath := filepath.Join(binDir, "gh")
	script := "#!/bin/sh\nif [ \"$1\" = \"pr\" ] && [ \"$2\" = \"view\" ]; then\n  echo OPEN\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(ghPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write gh stub: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	resolver := NewBranchStatusResolver(repo)
	got, err := resolver.Status(branchPath, branchName)
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if got != BranchStatusOpened {
		t.Fatalf("unexpected status: want %q got %q", BranchStatusOpened, got)
	}
}

func TestBranchStatusResolver_shouldReturnMerged_whenGhReportsMergedPR(t *testing.T) {
	const branchName = "feature/merged"
	repo, branchPath := initStatusTestRepo(t, branchName)
	path := filepath.Join(branchPath, "merged.txt")
	if err := os.WriteFile(path, []byte("merged"), 0o644); err != nil {
		t.Fatalf("write merged: %v", err)
	}
	runGitTestHelper(t, branchPath, "add", "merged.txt")
	runGitTestHelper(t, branchPath, "commit", "-m", "feat: merge work")
	runGitTestHelper(t, repo, "merge", "--ff-only", branchName)

	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	ghPath := filepath.Join(binDir, "gh")
	script := "#!/bin/sh\nif [ \"$1\" = \"pr\" ] && [ \"$2\" = \"view\" ]; then\n  echo MERGED\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(ghPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write gh stub: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	resolver := NewBranchStatusResolver(repo)
	got, err := resolver.Status(branchPath, branchName)
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if got != BranchStatusMerged {
		t.Fatalf("unexpected status: want %q got %q", BranchStatusMerged, got)
	}
}

func TestBranchStatus_Display_shouldReturnUppercase(t *testing.T) {
	if got := BranchStatusInProgress.Display(); got != "IN PROGRESS" {
		t.Fatalf("unexpected display: %q", got)
	}
	if got := BranchStatusMerged.Display(); got != "MERGED" {
		t.Fatalf("unexpected display: %q", got)
	}
	if got := BranchStatus("").Display(); got != "" {
		t.Fatalf("expected empty display, got %q", got)
	}
}

func initStatusTestRepo(t *testing.T, branch string) (string, string) {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitTestHelper(t, root, "init", "--initial-branch=main", "repo")
	runGitTestHelper(t, repo, "config", "user.email", "test@example.com")
	runGitTestHelper(t, repo, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("init"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGitTestHelper(t, repo, "add", ".")
	runGitTestHelper(t, repo, "commit", "-m", "init")
	branchPath := filepath.Join(root, "wt")
	runGitTestHelper(t, repo, "worktree", "add", branchPath, "-b", branch)
	return repo, branchPath
}

func runGitTestHelper(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", args, err, out)
	}
}
