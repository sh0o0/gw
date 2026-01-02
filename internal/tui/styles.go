package tui

import "github.com/charmbracelet/lipgloss"

var (
	primaryColor   = lipgloss.Color("#7D56F4")
	secondaryColor = lipgloss.Color("#5B4B8A")
	accentColor    = lipgloss.Color("#FF79C6")
	successColor   = lipgloss.Color("#50FA7B")
	warningColor   = lipgloss.Color("#FFB86C")
	errorColor     = lipgloss.Color("#FF5555")
	infoColor      = lipgloss.Color("#8BE9FD")
	mutedColor     = lipgloss.Color("#6272A4")
	textColor      = lipgloss.Color("#F8F8F2")
	dimTextColor   = lipgloss.Color("#888888")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Background(lipgloss.Color("#282A36")).
			Padding(0, 1).
			MarginBottom(1)

	tabStyle = lipgloss.NewStyle().
			Padding(0, 2)

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#282A36")).
			Background(primaryColor).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Padding(0, 2)

	listItemStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(textColor)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(0).
				Foreground(accentColor).
				Bold(true)

	currentWorktreeStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	primaryStyle = lipgloss.NewStyle().
			Foreground(infoColor).
			Italic(true)

	branchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BD93F9"))

	pathStyle = lipgloss.NewStyle().
			Foreground(dimTextColor).
			Italic(true)

	statusOpenStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	statusInProgressStyle = lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true)

	statusMergedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#BD93F9")).
				Bold(true)

	statusDraftStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(dimTextColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	successMsgStyle = lipgloss.NewStyle().
			Foreground(successColor)

	warningMsgStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(secondaryColor).
			Padding(1, 2)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(accentColor).
			Padding(1, 2).
			Background(lipgloss.Color("#282A36"))

	symlinkActiveStyle = lipgloss.NewStyle().
				Foreground(successColor)

	symlinkInactiveStyle = lipgloss.NewStyle().
				Foreground(dimTextColor)

	filterStyle = lipgloss.NewStyle().
			Foreground(infoColor).
			Bold(true)

	loadingStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	cursorStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)
)
