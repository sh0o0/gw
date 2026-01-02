package component

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	confirmModalStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("#FF5555")).
				Padding(1, 3).
				Background(lipgloss.Color("#282A36"))

	dangerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FF5555"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C"))

	messageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2"))

	detailStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Italic(true)

	cancelSelectedStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Bold(true).
				Foreground(lipgloss.Color("#282A36")).
				Background(lipgloss.Color("#6272A4"))

	deleteSelectedStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Bold(true).
				Foreground(lipgloss.Color("#282A36")).
				Background(lipgloss.Color("#FF5555"))

	unselectedBtnStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Foreground(lipgloss.Color("#6272A4"))
)

type ConfirmModal struct {
	Title     string
	Message   string
	Detail    string
	confirmed bool
	cancelled bool
	selected  int // 0 = cancel, 1 = confirm
}

func NewConfirmModal(title, message, detail string) ConfirmModal {
	return ConfirmModal{
		Title:    title,
		Message:  message,
		Detail:   detail,
		selected: 0,
	}
}

func (m ConfirmModal) Init() tea.Cmd {
	return nil
}

func (m ConfirmModal) Update(msg tea.Msg) (ConfirmModal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			m.selected = 0
		case "right", "l":
			m.selected = 1
		case "tab":
			m.selected = (m.selected + 1) % 2
		case "enter":
			if m.selected == 1 {
				m.confirmed = true
			} else {
				m.cancelled = true
			}
		case "esc", "n":
			m.cancelled = true
		case "y":
			m.confirmed = true
		}
	}
	return m, nil
}

func (m ConfirmModal) View() string {
	var b strings.Builder

	b.WriteString(dangerTitleStyle.Render("⚠ " + m.Title))
	b.WriteString("\n\n")
	b.WriteString(messageStyle.Render(m.Message))
	b.WriteString("\n\n")

	if m.Detail != "" {
		b.WriteString(detailStyle.Render(m.Detail))
		b.WriteString("\n\n")
	}

	var cancelBtn, confirmBtn string

	if m.selected == 0 {
		cancelBtn = cancelSelectedStyle.Render(" ✗ Cancel ")
		confirmBtn = unselectedBtnStyle.Render("Delete")
	} else {
		cancelBtn = unselectedBtnStyle.Render("Cancel")
		confirmBtn = deleteSelectedStyle.Render(" ✗ Delete ")
	}

	b.WriteString(cancelBtn + "    " + confirmBtn)
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("← → to select, Enter to confirm, Esc to cancel"))

	return confirmModalStyle.Render(b.String())
}

func (m ConfirmModal) Confirmed() bool {
	return m.confirmed
}

func (m ConfirmModal) Cancelled() bool {
	return m.cancelled
}
