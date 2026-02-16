package statusbar

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/paramon-tech/tgtui/internal/ui/common"
)

type Model struct {
	text  string
	mode  string
	width int
}

func New() Model {
	return Model{text: "Connecting...", mode: "NOR"}
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
	modeStyle := lipgloss.NewStyle().Bold(true)
	switch m.mode {
	case "INS":
		modeStyle = modeStyle.Foreground(common.ColorSecondary)
	default:
		modeStyle = modeStyle.Foreground(common.ColorPrimary)
	}
	mode := modeStyle.Render(m.mode)
	text := lipgloss.NewStyle().Foreground(common.ColorMuted).Render(m.text)
	return lipgloss.NewStyle().Width(m.width).Padding(0, 1).Render(mode + "  " + text)
}

func (m Model) SetSize(w int) Model {
	m.width = w
	return m
}

func (m Model) SetMode(mode string) Model {
	m.mode = mode
	return m
}
