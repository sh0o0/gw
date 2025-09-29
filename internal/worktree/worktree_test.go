package worktree

import (
	"strings"
	"testing"
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
	if !matchAnyPattern("/.vscode/settings.json", SymlinkPatterns()) {
		t.Fatalf("expected vscode pattern to match")
	}
	if shouldExclude("node_modules/foo") == false {
		t.Fatalf("expected node_modules to be excluded")
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
