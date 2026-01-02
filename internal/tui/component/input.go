package component

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("170")).
			Padding(1, 2)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			MarginBottom(1)

	buttonStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("245"))

	activeButtonStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Bold(true).
				Foreground(lipgloss.Color("170"))
)

type InputModal struct {
	Title       string
	Placeholder string
	Input       textinput.Model
	Focused     bool
	confirmed   bool
	cancelled   bool
}

func NewInputModal(title, placeholder string) InputModal {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40

	return InputModal{
		Title:       title,
		Placeholder: placeholder,
		Input:       ti,
		Focused:     true,
	}
}

func (m InputModal) Init() tea.Cmd {
	return textinput.Blink
}

func (m InputModal) Update(msg tea.Msg) (InputModal, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.Input.Value() != "" {
				m.confirmed = true
			}
			return m, nil
		case "esc":
			m.cancelled = true
			return m, nil
		}
	}

	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

func (m InputModal) View() string {
	var b strings.Builder

	b.WriteString(modalTitleStyle.Render(m.Title))
	b.WriteString("\n\n")
	b.WriteString(m.Input.View())
	b.WriteString("\n\n")

	cancelBtn := buttonStyle.Render("[Esc] Cancel")
	confirmBtn := activeButtonStyle.Render("[Enter] Create")
	b.WriteString(cancelBtn + "  " + confirmBtn)

	return modalStyle.Render(b.String())
}

func (m InputModal) Value() string {
	return m.Input.Value()
}

func (m InputModal) Confirmed() bool {
	return m.confirmed
}

func (m InputModal) Cancelled() bool {
	return m.cancelled
}
