package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170"))

	tabStyle = lipgloss.NewStyle().
			Padding(0, 2)

	activeTabStyle = tabStyle.
			Bold(true).
			Foreground(lipgloss.Color("170")).
			Underline(true)

	inactiveTabStyle = tabStyle.
				Foreground(lipgloss.Color("245"))

	listItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("170")).
				Bold(true)

	primaryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)

	statusOpenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	statusInProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))

	statusMergedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("141"))

	statusDraftStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238"))
)
