package statusbar

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/paramon-tech/tgtui/internal/ui/common"
)

type Model struct {
	text  string
	width int
}

func New() Model {
	return Model{text: "Connecting..."}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.StatusMsg:
		m.text = msg.Text
	case common.AuthorizedMsg:
		m.text = "Connected"
	case common.NeedAuthMsg:
		m.text = "Authentication required"
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

func (m Model) View() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Foreground(common.ColorMuted).
		Padding(0, 1)

	return style.Render(m.text)
}

func (m Model) SetSize(w int) Model {
	m.width = w
	return m
}
