package chatview

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/paramon-tech/tgtui/internal/telegram"
	"github.com/paramon-tech/tgtui/internal/ui/common"
)

type Model struct {
	chat          *telegram.Chat
	messages      []telegram.Message
	input         string
	tg            *telegram.Client
	focused       bool
	width, height int
	scrollOffset  int
	inputFocused  bool
}

func New(tg *telegram.Client) Model {
	return Model{
		tg:           tg,
		inputFocused: true,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.HistoryLoadedMsg:
		if m.chat != nil && msg.ChatID == m.chat.ID {
			m.messages = msg.Messages
			m.scrollOffset = 0
		}

	case common.NewMessageMsg:
		if m.chat != nil && msg.Message.ChatID == m.chat.ID {
			m.messages = append(m.messages, msg.Message)
			m.scrollOffset = 0
		}

	case common.MessageSendErrorMsg:
		return m, func() tea.Msg {
			return common.StatusMsg{Text: "Send failed: " + msg.Err.Error()}
		}

	case tea.KeyMsg:
		if !m.focused || m.chat == nil {
			return m, nil
		}

		if m.inputFocused && m.chat.Type != telegram.ChatTypeChannel {
			return m.handleInputKey(msg)
		}
		return m.handleViewportKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m Model) handleInputKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inputFocused = false
		return m, nil
	case "pgup":
		m.inputFocused = false
		return m, nil
	}

	switch msg.Type {
	case tea.KeyEnter:
		text := strings.TrimSpace(m.input)
		if text == "" {
			return m, nil
		}
		m.input = ""
		chat := *m.chat
		tg := m.tg
		return m, func() tea.Msg {
			return tg.SendMessage(chat, text)()
		}

	case tea.KeyBackspace:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}

	case tea.KeyRunes:
		m.input += string(msg.Runes)

	case tea.KeySpace:
		m.input += " "
	}

	return m, nil
}

func (m Model) handleViewportKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.scrollOffset < len(m.messages)-1 {
			m.scrollOffset++
		}
	case "down", "j":
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
	case "pgup":
		m.scrollOffset += 10
		max := len(m.messages) - 1
		if max < 0 {
			max = 0
		}
		if m.scrollOffset > max {
			m.scrollOffset = max
		}
	case "pgdown":
		m.scrollOffset -= 10
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
	case "enter":
		if m.chat.Type != telegram.ChatTypeChannel {
			m.inputFocused = true
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.chat == nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			common.StyleMuted.Render("Select a chat to start messaging"))
	}

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(common.ColorPrimary).
		Width(m.width).
		Padding(0, 1)
	title := titleStyle.Render(m.chat.Title)

	// Calculate available height
	inputHeight := 0
	if m.chat.Type != telegram.ChatTypeChannel {
		inputHeight = 1
	}
	msgHeight := m.height - 1 - inputHeight // 1 for title

	// Messages
	msgView := m.renderMessages(msgHeight)

	// Input
	var inputView string
	if m.chat.Type != telegram.ChatTypeChannel {
		inputView = m.renderInput()
	}

	parts := []string{title, msgView}
	if inputView != "" {
		parts = append(parts, inputView)
	}

	return strings.Join(parts, "\n")
}

func (m Model) renderMessages(height int) string {
	if height <= 0 {
		return ""
	}

	if len(m.messages) == 0 {
		return lipgloss.Place(m.width, height, lipgloss.Center, lipgloss.Center,
			common.StyleMuted.Render("No messages"))
	}

	var lines []string
	for _, msg := range m.messages {
		line := m.renderMessage(msg)
		lines = append(lines, line)
	}

	// Apply scroll offset: show messages from end
	totalLines := len(lines)
	end := totalLines - m.scrollOffset
	if end < 0 {
		end = 0
	}
	start := end - height
	if start < 0 {
		start = 0
	}

	visible := lines[start:end]

	// Pad with empty lines at top if needed
	for len(visible) < height {
		visible = append([]string{""}, visible...)
	}

	return strings.Join(visible, "\n")
}

func (m Model) renderMessage(msg telegram.Message) string {
	ts := time.Unix(int64(msg.Date), 0).Format("15:04")
	timestamp := common.StyleTimestamp.Render("[" + ts + "]")

	var sender string
	if msg.Out {
		sender = common.StyleSenderSelf.Render("You")
	} else if msg.Sender != "" {
		sender = common.StyleSender.Render(msg.Sender)
	} else {
		sender = common.StyleSender.Render("Unknown")
	}

	text := msg.Text
	if text == "" {
		text = common.StyleMuted.Render("[non-text message]")
	}

	return fmt.Sprintf(" %s %s: %s", timestamp, sender, text)
}

func (m Model) renderInput() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1)

	prefix := common.StyleMuted.Render("> ")
	cursor := ""
	if m.focused && m.inputFocused {
		cursor = "█"
	}

	return style.Render(prefix + m.input + cursor)
}

func (m Model) SetChat(chat *telegram.Chat) Model {
	m.chat = chat
	m.messages = nil
	m.input = ""
	m.scrollOffset = 0
	m.inputFocused = true
	return m
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	return m
}

func (m Model) SetFocus(focused bool) Model {
	m.focused = focused
	if focused {
		m.inputFocused = true
	}
	return m
}

func (m Model) Focused() bool {
	return m.focused
}
