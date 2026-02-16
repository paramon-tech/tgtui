package chatlist

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/paramon-tech/tgtui/internal/telegram"
	"github.com/paramon-tech/tgtui/internal/ui/common"
)

type Model struct {
	chats         []telegram.Chat
	cursor        int
	offset        int
	focused       bool
	width, height int
}

func New() Model {
	return Model{focused: true}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.DialogsLoadedMsg:
		m.chats = msg.Chats
		m.cursor = 0
		m.offset = 0

	case common.NewMessageMsg:
		m.updateOnNewMessage(msg.Message)

	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}
		case "down", "j":
			if m.cursor < len(m.chats)-1 {
				m.cursor++
				visible := m.visibleCount()
				if m.cursor >= m.offset+visible {
					m.offset = m.cursor - visible + 1
				}
			}
		case "enter":
			if m.cursor < len(m.chats) {
				chat := m.chats[m.cursor]
				return m, func() tea.Msg {
					return common.ChatSelectedMsg{Chat: chat}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m *Model) updateOnNewMessage(msg telegram.Message) {
	for i, c := range m.chats {
		if c.ID == msg.ChatID {
			if !msg.Out {
				m.chats[i].UnreadCount++
			}
			m.chats[i].LastMessage = &telegram.Message{
				ID:       msg.ID,
				ChatID:   msg.ChatID,
				SenderID: msg.SenderID,
				Sender:   msg.Sender,
				Text:     msg.Text,
				Date:     msg.Date,
				Out:      msg.Out,
			}
			// Move to top
			chat := m.chats[i]
			copy(m.chats[1:i+1], m.chats[0:i])
			m.chats[0] = chat
			return
		}
	}
}

func (m Model) View() string {
	if len(m.chats) == 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			common.StyleMuted.Render("Loading chats..."))
	}

	var lines []string
	visible := m.visibleCount()

	for i := m.offset; i < len(m.chats) && i < m.offset+visible; i++ {
		chat := m.chats[i]
		line := m.renderChat(chat, i == m.cursor)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")

	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height)

	return style.Render(content)
}

func (m Model) renderChat(chat telegram.Chat, selected bool) string {
	width := m.width - 2
	if width < 4 {
		width = 4
	}

	var prefix string
	switch chat.Type {
	case telegram.ChatTypeGroup:
		prefix = "#"
	case telegram.ChatTypeChannel:
		prefix = ">"
	default:
		prefix = " "
	}

	name := truncate(chat.Title, width-8)

	var unread string
	if chat.UnreadCount > 0 {
		unread = common.StyleUnread.Render(fmt.Sprintf(" (%d)", chat.UnreadCount))
	}

	line := fmt.Sprintf(" %s %s%s", prefix, name, unread)

	if selected && m.focused {
		return common.StyleSelected.Render(">" + line)
	}
	return " " + line
}

func (m Model) visibleCount() int {
	if m.height <= 0 {
		return 20
	}
	return m.height
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	return m
}

func (m Model) SetFocus(focused bool) Model {
	m.focused = focused
	return m
}

func (m Model) SelectedChat() (telegram.Chat, bool) {
	if m.cursor < len(m.chats) {
		return m.chats[m.cursor], true
	}
	return telegram.Chat{}, false
}

func (m Model) Focused() bool {
	return m.focused
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "…"
}
