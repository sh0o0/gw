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
	symlinkActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#50FA7B")).
				Bold(true)

	symlinkInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	targetStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Italic(true)

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF79C6")).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BD93F9")).
			Bold(true)

	notLinkedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C"))
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

	if selected {
		b.WriteString(cursorStyle.Render("❯ "))
	} else {
		b.WriteString("  ")
	}

	pathDisplay := item.Path
	if item.IsSymlink {
		if selected {
			b.WriteString(selectedStyle.Render(pathDisplay))
		} else {
			b.WriteString(symlinkActiveStyle.Render(pathDisplay))
		}
	} else {
		if selected {
			b.WriteString(selectedStyle.Render(pathDisplay))
		} else {
			b.WriteString(symlinkInactiveStyle.Render(pathDisplay))
		}
	}

	padding := maxPathLen - len(item.Path) + 2
	b.WriteString(strings.Repeat(" ", padding))

	if item.IsSymlink {
		b.WriteString(targetStyle.Render("🔗 → " + item.Target))
	} else if item.Target != "" {
		b.WriteString(notLinkedStyle.Render("○ not linked"))
	}

	return b.String()
}
