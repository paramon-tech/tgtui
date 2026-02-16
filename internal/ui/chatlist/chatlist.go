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
	activeChatID  int64
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
				Media:    msg.Media,
			}
			// Move to top and adjust cursor/offset
			chat := m.chats[i]
			copy(m.chats[1:i+1], m.chats[0:i])
			m.chats[0] = chat

			if m.cursor == i {
				// Cursor was on the moved chat — follow it to top
				m.cursor = 0
				m.offset = 0
			} else if m.cursor < i {
				// Cursor is above the moved chat — items shifted down
				m.cursor++
				if m.cursor >= m.offset+m.visibleCount() {
					m.offset++
				}
			}
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

	// Pad to fixed height
	for len(lines) < m.height {
		lines = append(lines, "")
	}

	return strings.Join(lines[:m.height], "\n")
}

func (m Model) renderChat(chat telegram.Chat, selected bool) string {
	width := m.width - 2
	if width < 4 {
		width = 4
	}

	var typePrefix string
	switch chat.Type {
	case telegram.ChatTypeGroup:
		typePrefix = "#"
	case telegram.ChatTypeChannel:
		typePrefix = ">"
	default:
		typePrefix = " "
	}

	name := truncate(chat.Title, width-8)

	var unread string
	if chat.UnreadCount > 0 {
		unread = common.StyleUnread.Render(fmt.Sprintf(" (%d)", chat.UnreadCount))
	}

	line := fmt.Sprintf(" %s %s%s", typePrefix, name, unread)
	isActive := chat.ID == m.activeChatID && m.activeChatID != 0

	var marker string
	if selected && m.focused {
		// Cursor + focused: bold primary with ">"
		return lipgloss.NewStyle().MaxWidth(m.width).Render(common.StyleSelected.Render(">" + line))
	} else if selected {
		// Cursor + unfocused: muted ">" marker
		marker = common.StyleMuted.Render(">")
		return lipgloss.NewStyle().MaxWidth(m.width).Render(marker + line)
	} else if isActive {
		// Active chat (not cursor): primary "│" marker
		marker = lipgloss.NewStyle().Foreground(common.ColorPrimary).Render("│")
		return lipgloss.NewStyle().MaxWidth(m.width).Render(marker + line)
	}
	// Normal: space marker
	return lipgloss.NewStyle().MaxWidth(m.width).Render(" " + line)
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

func (m Model) SetActiveChat(id int64) Model {
	m.activeChatID = id
	return m
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
