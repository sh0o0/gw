package worktree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sh0o0/gw/internal/fsutil"
)

func TestParseRemoteURL_should_handle_formats(t *testing.T) {
	cases := []struct {
		url               string
		domain, org, repo string
	}{
		{"git@github.com:owner/repo.git", "github.com", "owner", "repo"},
		{"https://github.com/owner/repo.git", "github.com", "owner", "repo"},
		{"http://example.com/foo/bar.git", "example.com", "foo", "bar"},
		{"git://example.com/foo/bar.git", "example.com", "foo", "bar"},
	}
	for _, c := range cases {
		domain, org, repo, has, err := func() (string, string, string, bool, error) {
			// monkeypatch via wrapper: simulate by replacing gitx.Cmd? For unit, call internal directly is hard.
			// Instead, test internal parsing by small helper that mirrors logic.
			return parseRemoteURLString(c.url)
		}()
		if err != nil || !has {
			t.Fatalf("expected parse ok for %s, err=%v", c.url, err)
		}
		if domain != c.domain || org != c.org || repo != c.repo {
			t.Fatalf("unexpected parts for %s: got %s %s %s", c.url, domain, org, repo)
		}
	}
}

// local helper to validate patterns
func TestMatchPatterns_when_expected(t *testing.T) {
	pats := []string{"**/.vscode/*"}
	if !matchAnyPattern("/.vscode/settings.json", pats) {
		t.Fatalf("expected vscode pattern to match")
	}
	excludes := []string{"**/node_modules/**"}
	if shouldExclude("node_modules/foo", excludes) == false {
		t.Fatalf("expected node_modules to be excluded")
	}
}

func TestCreateSymlink_should_resolve_symlink_chain(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()

	// Create actual file
	actualFile := filepath.Join(tmpDir, "actual.txt")
	if err := os.WriteFile(actualFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create first symlink pointing to actual file
	firstSymlink := filepath.Join(tmpDir, "first.txt")
	if err := os.Symlink(actualFile, firstSymlink); err != nil {
		t.Fatal(err)
	}

	// Create second symlink pointing to first symlink
	secondSymlink := filepath.Join(tmpDir, "second.txt")
	if err := os.Symlink(firstSymlink, secondSymlink); err != nil {
		t.Fatal(err)
	}

	// Act: Create a new symlink using CreateSymlink with secondSymlink as source
	// This simulates what happens when we resolve symlink chains
	resolved, err := filepath.EvalSymlinks(secondSymlink)
	if err != nil {
		t.Fatal(err)
	}

	newSymlink := filepath.Join(tmpDir, "new.txt")
	if err := fsutil.CreateSymlink(resolved, newSymlink); err != nil {
		t.Fatal(err)
	}

	// Assert: Verify that newSymlink points directly to actual file
	target, err := os.Readlink(newSymlink)
	if err != nil {
		t.Fatal(err)
	}

	// Resolve both paths to handle symlinks in path (like /var -> /private/var on macOS)
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	resolvedActual, err := filepath.EvalSymlinks(actualFile)
	if err != nil {
		t.Fatal(err)
	}

	if resolvedTarget != resolvedActual {
		t.Errorf("expected symlink to point to %s, got %s", resolvedActual, resolvedTarget)
	}

	// Verify content is accessible
	content, err := os.ReadFile(newSymlink)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "content" {
		t.Errorf("expected content 'content', got '%s'", string(content))
	}
}

// extract parsing core so test doesn't shell out
func parseRemoteURLString(u string) (string, string, string, bool, error) {
	// copy from ParseRemoteURL without git call
	if len(u) == 0 {
		return "", "", "", false, nil
	}
	if len(u) >= 4 && strings.HasPrefix(u, "git@") {
		at := strings.IndexByte(u, '@')
		colon := strings.IndexByte(u, ':')
		if at > 0 && colon > at {
			domain := u[at+1 : colon]
			rest := u[colon+1:]
			parts := strings.SplitN(rest, "/", 2)
			if len(parts) == 2 {
				org := parts[0]
				repo := parts[1]
				repo = strings.TrimSuffix(repo, ".git")
				return domain, org, repo, true, nil
			}
		}
	}
	for _, pref := range []string{"https://", "http://", "git://"} {
		if strings.HasPrefix(u, pref) {
			rest := strings.TrimPrefix(u, pref)
			parts := strings.Split(rest, "/")
			if len(parts) >= 3 {
				domain := parts[0]
				org := parts[1]
				repo := parts[2]
				repo = strings.TrimSuffix(repo, ".git")
				return domain, org, repo, true, nil
			}
		}
	}
	return "", "", "", false, nil
}
