package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	New       key.Binding
	Add       key.Binding
	Delete    key.Binding
	Link      key.Binding
	Unlink    key.Binding
	Sync      key.Binding
	Search    key.Binding
	Help      key.Binding
	Tab1      key.Binding
	Tab2      key.Binding
	Escape    key.Binding
	Quit      key.Binding
	ForceQuit key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "switch"),
		),
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new worktree"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add existing branch"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Link: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "link"),
		),
		Unlink: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "unlink"),
		),
		Sync: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sync all"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Tab1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "worktrees"),
		),
		Tab2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "symlinks"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "force quit"),
		),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.New, k.Delete, k.Search, k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.New, k.Add, k.Delete},
		{k.Search, k.Tab1, k.Tab2},
		{k.Help, k.Escape, k.Quit},
	}
}
