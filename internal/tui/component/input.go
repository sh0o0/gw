package component

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 3).
			Background(lipgloss.Color("#282A36"))

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF79C6")).
			MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4"))

	buttonStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("#6272A4"))

	activeButtonStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Bold(true).
				Foreground(lipgloss.Color("#282A36")).
				Background(lipgloss.Color("#50FA7B"))

	cancelButtonStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Foreground(lipgloss.Color("#FF5555"))
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

	b.WriteString(modalTitleStyle.Render("✨ " + m.Title))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("Branch name:"))
	b.WriteString("\n")
	b.WriteString(m.Input.View())
	b.WriteString("\n\n")

	cancelBtn := cancelButtonStyle.Render("✗ Esc")
	confirmBtn := activeButtonStyle.Render(" ✓ Enter ")
	b.WriteString(cancelBtn + "    " + confirmBtn)

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
