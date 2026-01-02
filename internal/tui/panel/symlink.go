package panel

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sh0o0/gw/internal/gitx"
	"github.com/sh0o0/gw/internal/worktree"
)

var (
	symlinkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("45"))

	targetStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
)

type SymlinkItem struct {
	Path       string
	Target     string
	IsSymlink  bool
	IsLinkable bool
}

func LoadSymlinks(wtPath string) ([]SymlinkItem, error) {
	primaryPath, err := gitx.Root(wtPath)
	if err != nil {
		return nil, err
	}

	files, err := worktree.GitIgnoredFiles(primaryPath)
	if err != nil {
		return nil, err
	}

	pats := worktree.SymlinkPatterns(primaryPath)
	excludes := worktree.ExcludePatterns(primaryPath)

	var items []SymlinkItem
	for _, f := range files {
		if shouldExclude(f, excludes) {
			continue
		}
		if !matchAnyPattern("/"+f, pats) {
			continue
		}

		srcPath := filepath.Join(primaryPath, f)
		dstPath := filepath.Join(wtPath, f)

		item := SymlinkItem{
			Path:       f,
			IsLinkable: true,
		}

		if info, err := os.Lstat(dstPath); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				target, _ := os.Readlink(dstPath)
				item.IsSymlink = true
				item.Target = target
			}
		} else {
			if _, err := os.Lstat(srcPath); err == nil {
				item.Target = srcPath
			}
		}

		items = append(items, item)
	}

	return items, nil
}

func shouldExclude(path string, excludes []string) bool {
	for _, p := range excludes {
		if matched, _ := filepath.Match(p, path); matched {
			return true
		}
		if strings.Contains(p, "**") {
			if matchGlob(p, path) {
				return true
			}
		}
	}
	return false
}

func matchAnyPattern(path string, patterns []string) bool {
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, path); matched {
			return true
		}
		if strings.Contains(p, "**") {
			if matchGlob(p, path) {
				return true
			}
		}
	}
	return false
}

func matchGlob(pattern, path string) bool {
	parts := strings.Split(pattern, "**")
	if len(parts) == 2 {
		prefix := parts[0]
		suffix := parts[1]
		return strings.HasPrefix(path, prefix) && strings.HasSuffix(path, suffix)
	}
	return false
}

func RenderSymlinkItem(idx int, item SymlinkItem, selected bool, maxPathLen int) string {
	var b strings.Builder

	prefix := "  "
	if selected {
		prefix = "> "
	}

	b.WriteString(prefix)

	if item.IsSymlink {
		b.WriteString(symlinkStyle.Render(item.Path))
	} else {
		b.WriteString(item.Path)
	}

	padding := maxPathLen - len(item.Path) + 2
	b.WriteString(strings.Repeat(" ", padding))

	if item.IsSymlink {
		b.WriteString(targetStyle.Render("-> " + item.Target))
	} else if item.Target != "" {
		b.WriteString(targetStyle.Render("(not linked)"))
	}

	return b.String()
}
