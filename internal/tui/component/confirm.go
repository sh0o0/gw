package component

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	dangerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
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

	b.WriteString(dangerStyle.Render(m.Title))
	b.WriteString("\n\n")
	b.WriteString(m.Message)
	b.WriteString("\n\n")

	if m.Detail != "" {
		b.WriteString(warningStyle.Render(m.Detail))
		b.WriteString("\n\n")
	}

	cancelBtn := buttonStyle.Render("[Cancel]")
	confirmBtn := buttonStyle.Render("[Delete]")

	if m.selected == 0 {
		cancelBtn = activeButtonStyle.Render("[Cancel]")
	} else {
		confirmBtn = dangerStyle.Bold(true).Padding(0, 2).Render("[Delete]")
	}

	b.WriteString(cancelBtn + "  " + confirmBtn)

	return modalStyle.Render(b.String())
}

func (m ConfirmModal) Confirmed() bool {
	return m.confirmed
}

func (m ConfirmModal) Cancelled() bool {
	return m.cancelled
}
