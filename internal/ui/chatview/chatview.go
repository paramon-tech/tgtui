package chatview

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/paramon-tech/tgtui/internal/format"
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
	cursor        int
	expandedMsgID int
	// History pagination
	loadingOlder bool
	noMoreHistory bool
	// Photo thumbnail cache
	photoCache   map[int]string // msgID → rendered half-block string
	photoLines   map[int]int    // msgID → line count of rendered image
	photoLoading map[int]bool   // msgID → currently downloading
	// File download state
	fileSaving map[int]bool // msgID → currently saving to disk
	// Visual/selection mode
	selecting bool
	selected  map[int]bool // selected message IDs
	// Search mode
	searching      bool
	searchQuery    string
	searchResults  []telegram.Message // messages returned by search
	searchActive   bool               // true when showing search results
}

func New(tg *telegram.Client) Model {
	return Model{
		tg:            tg,
		inputFocused:  true,
		expandedMsgID: -1,
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
			m.cursor = len(m.messages) - 1
			m.expandedMsgID = -1
			m.loadingOlder = false
			m.noMoreHistory = false
		}

	case common.OlderHistoryLoadedMsg:
		if m.chat != nil && msg.ChatID == m.chat.ID {
			m.loadingOlder = false
			if len(msg.Messages) == 0 {
				m.noMoreHistory = true
			} else {
				// Prepend older messages, adjust cursor to keep position
				m.cursor += len(msg.Messages)
				m.messages = append(msg.Messages, m.messages...)
				m.ensureCursorVisible()
			}
		}

	case common.OlderHistoryErrorMsg:
		m.loadingOlder = false

	case common.NewMessageMsg:
		if m.chat != nil && msg.Message.ChatID == m.chat.ID {
			m.messages = append(m.messages, msg.Message)
			m.scrollOffset = 0
			m.expandedMsgID = -1
		}

	case common.ReactionsUpdatedMsg:
		if m.chat != nil && msg.ChatID == m.chat.ID {
			for i := range m.messages {
				if m.messages[i].ID == msg.MsgID {
					m.messages[i].Reactions = msg.Reactions
					break
				}
			}
		}

	case common.SearchResultMsg:
		if m.chat != nil && msg.ChatID == m.chat.ID {
			m.searchResults = msg.Messages
			m.searchActive = true
			m.scrollOffset = 0
			m.expandedMsgID = -1
			if len(msg.Messages) > 0 {
				m.cursor = len(msg.Messages) - 1
			} else {
				m.cursor = -1
			}
			return m, func() tea.Msg {
				return common.StatusMsg{Text: fmt.Sprintf("Found %d result(s) for \"%s\" — Esc to go back", len(msg.Messages), msg.Query)}
			}
		}

	case common.SearchErrorMsg:
		return m, func() tea.Msg {
			return common.StatusMsg{Text: "Search failed: " + msg.Err.Error()}
		}

	case common.HistoryErrorMsg:
		return m, func() tea.Msg {
			return common.StatusMsg{Text: "Failed to load messages: " + msg.Err.Error()}
		}

	case common.MessageSendErrorMsg:
		return m, func() tea.Msg {
			return common.StatusMsg{Text: "Send failed: " + msg.Err.Error()}
		}

	case common.MessageSentMsg:
		if m.chat != nil && msg.ChatID == m.chat.ID {
			tg := m.tg
			chat := *m.chat
			return m, func() tea.Msg {
				return tg.FetchHistory(chat)()
			}
		}

	case common.DownloadPhotoMsg:
		// Use half-block rendering for inline TUI display.
		// Kitty/iTerm2/Sixel protocols render on a separate graphics layer
		// that persists across Bubble Tea redraws and overlaps text.
		rendered, lines, err := format.RenderImageHalfBlock(msg.Data, m.photoMaxWidth(), m.photoMaxHeight())
		if err == nil {
			m.initPhotoCaches()
			m.photoCache[msg.MessageID] = rendered
			m.photoLines[msg.MessageID] = lines
		}
		delete(m.photoLoading, msg.MessageID)
		m.ensureCursorVisible()

	case common.DownloadPhotoErrorMsg:
		delete(m.photoLoading, msg.MessageID)

	case common.SaveFileMsg:
		delete(m.fileSaving, msg.MessageID)
		return m, func() tea.Msg {
			return common.StatusMsg{Text: "Saved: " + msg.Path}
		}

	case common.SaveFileErrorMsg:
		delete(m.fileSaving, msg.MessageID)
		return m, func() tea.Msg {
			return common.StatusMsg{Text: "Download failed: " + msg.Err.Error()}
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
	case "pgup":
		m.inputFocused = false
		m.clampCursor()
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

func (m Model) handleSelectionKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}
	case "down", "j":
		if m.cursor < len(m.messages)-1 {
			m.cursor++
			m.ensureCursorVisible()
		}
	case " ":
		if m.cursor >= 0 && m.cursor < len(m.messages) {
			msgID := m.messages[m.cursor].ID
			if m.selected[msgID] {
				delete(m.selected, msgID)
			} else {
				if m.selected == nil {
					m.selected = make(map[int]bool)
				}
				m.selected[msgID] = true
			}
		}
	case "f":
		if len(m.selected) == 0 {
			return m, func() tea.Msg {
				return common.StatusMsg{Text: "No messages selected"}
			}
		}
		var ids []int
		for _, msg := range m.messages {
			if m.selected[msg.ID] {
				ids = append(ids, msg.ID)
			}
		}
		chat := *m.chat
		return m, func() tea.Msg {
			return common.ForwardRequestMsg{
				FromChat:   chat,
				MessageIDs: ids,
			}
		}
	case "esc":
		m.selecting = false
		m.selected = nil
	case "pgup":
		pageSize := m.msgAreaHeight()
		if pageSize < 1 {
			pageSize = 1
		}
		m.cursor -= pageSize
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.ensureCursorVisible()
	case "pgdown":
		pageSize := m.msgAreaHeight()
		if pageSize < 1 {
			pageSize = 1
		}
		m.cursor += pageSize
		if m.cursor >= len(m.messages) {
			m.cursor = len(m.messages) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.ensureCursorVisible()
	}
	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		query := strings.TrimSpace(m.searchQuery)
		if query == "" {
			m.searching = false
			m.searchQuery = ""
			return m, nil
		}
		m.searching = false
		chat := *m.chat
		tg := m.tg
		return m, func() tea.Msg {
			return tg.SearchHistory(chat, query)()
		}

	case tea.KeyEscape:
		m.searching = false
		m.searchQuery = ""
		return m, nil

	case tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}

	case tea.KeyRunes:
		m.searchQuery += string(msg.Runes)

	case tea.KeySpace:
		m.searchQuery += " "
	}

	return m, nil
}

func (m Model) handleViewportKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.searching {
		return m.handleSearchKey(msg)
	}
	if m.selecting {
		return m.handleSelectionKey(msg)
	}

	msgs := m.activeMessages()

	switch msg.String() {
	case "esc":
		if m.searchActive {
			m.searchActive = false
			m.searchResults = nil
			m.scrollOffset = 0
			m.cursor = len(m.messages) - 1
			m.ensureCursorVisible()
			return m, func() tea.Msg {
				return common.StatusMsg{Text: ""}
			}
		}
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}
		return m.maybeLoadOlder()
	case "down", "j":
		if m.cursor < len(msgs)-1 {
			m.cursor++
			m.ensureCursorVisible()
		}
	case "enter":
		if m.cursor >= 0 && m.cursor < len(msgs) {
			curMsg := msgs[m.cursor]
			msgID := curMsg.ID
			if m.expandedMsgID == msgID {
				m.expandedMsgID = -1
			} else {
				m.expandedMsgID = msgID
				// Trigger photo download if applicable
				if curMsg.Media != nil && curMsg.Media.Type == telegram.MediaPhoto && curMsg.Media.PhotoThumbSize != "" {
					if !m.photoLoading[msgID] && m.photoCache[msgID] == "" {
						m.initPhotoCaches()
						m.photoLoading[msgID] = true
						tgClient := m.tg
						info := curMsg.Media
						m.ensureCursorVisible()
						return m, func() tea.Msg {
							return tgClient.DownloadPhoto(msgID, info)()
						}
					}
				}
			}
			m.ensureCursorVisible()
		}
	case "pgup":
		pageSize := m.msgAreaHeight()
		if pageSize < 1 {
			pageSize = 1
		}
		m.cursor -= pageSize
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.ensureCursorVisible()
		return m.maybeLoadOlder()
	case "pgdown":
		pageSize := m.msgAreaHeight()
		if pageSize < 1 {
			pageSize = 1
		}
		m.cursor += pageSize
		if m.cursor >= len(msgs) {
			m.cursor = len(msgs) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.ensureCursorVisible()
	case "i":
		if m.chat.Type != telegram.ChatTypeChannel && !m.searchActive {
			m.inputFocused = true
		}
	case "v":
		if m.searchActive {
			return m, nil
		}
		m.selecting = true
		if m.selected == nil {
			m.selected = make(map[int]bool)
		}
		if m.cursor >= 0 && m.cursor < len(msgs) {
			m.selected[msgs[m.cursor].ID] = true
		}
	case "/":
		m.searching = true
		m.searchQuery = ""
		return m, nil
	case "D":
		if m.cursor >= 0 && m.cursor < len(msgs) {
			curMsg := msgs[m.cursor]
			if curMsg.Media != nil && m.isDownloadable(curMsg.Media) && !m.fileSaving[curMsg.ID] {
				destPath := m.downloadPath(curMsg.Media)
				if m.fileSaving == nil {
					m.fileSaving = make(map[int]bool)
				}
				m.fileSaving[curMsg.ID] = true
				tgClient := m.tg
				info := curMsg.Media
				msgID := curMsg.ID
				return m, tea.Batch(
					func() tea.Msg {
						return common.StatusMsg{Text: "Downloading to " + destPath + "..."}
					},
					func() tea.Msg {
						return tgClient.DownloadToFile(msgID, info, destPath)()
					},
				)
			}
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
		MaxWidth(m.width).
		Padding(0, 1)
	title := titleStyle.Render(m.chat.Title)

	// Calculate available height
	inputHeight := 0
	if m.chat.Type != telegram.ChatTypeChannel {
		inputHeight = 1
	}
	searchHeight := 0
	if m.searching {
		searchHeight = 1
	}
	msgHeight := m.height - 1 - inputHeight - searchHeight // 1 for title

	// Messages
	msgView := m.renderMessages(msgHeight)

	// Input
	var inputView string
	if m.chat.Type != telegram.ChatTypeChannel {
		inputView = m.renderInput()
	}

	// Search bar
	var searchView string
	if m.searching {
		searchView = m.renderSearchInput()
	}

	parts := []string{title, msgView}
	if inputView != "" {
		parts = append(parts, inputView)
	}
	if searchView != "" {
		parts = append(parts, searchView)
	}

	result := strings.Join(parts, "\n")

	// Ensure exact height by padding or truncating
	resultLines := strings.Split(result, "\n")
	for len(resultLines) < m.height {
		resultLines = append(resultLines, "")
	}
	return strings.Join(resultLines[:m.height], "\n")
}

func (m Model) renderMessages(height int) string {
	if height <= 0 {
		return ""
	}

	msgs := m.activeMessages()
	if len(msgs) == 0 {
		if m.searchActive {
			return lipgloss.Place(m.width, height, lipgloss.Center, lipgloss.Center,
				common.StyleMuted.Render("No results"))
		}
		return lipgloss.Place(m.width, height, lipgloss.Center, lipgloss.Center,
			common.StyleMuted.Render("No messages"))
	}

	// Build all visual lines
	var allLines []string
	if m.loadingOlder {
		allLines = append(allLines, common.StyleMuted.Render("  Loading older messages..."))
	}
	for i, msg := range msgs {
		isSelected := (i == m.cursor)
		isExpanded := msg.ID == m.expandedMsgID
		lines := m.renderMessageLines(msg, isSelected, isExpanded)
		allLines = append(allLines, lines...)
	}

	// Apply scroll offset: show from bottom
	totalLines := len(allLines)
	end := totalLines - m.scrollOffset
	if end < 0 {
		end = 0
	}
	if end > totalLines {
		end = totalLines
	}
	start := end - height
	if start < 0 {
		start = 0
	}

	visible := allLines[start:end]

	// Pad with empty lines at top if needed
	for len(visible) < height {
		visible = append([]string{""}, visible...)
	}

	return strings.Join(visible, "\n")
}

func (m Model) renderMessageLines(msg telegram.Message, isSelected, isExpanded bool) []string {
	ts := time.Unix(int64(msg.Date), 0).Format("Mon 02/01/2006 15:04")
	timestamp := common.StyleTimestamp.Render("[" + ts + "]")

	var sender string
	if msg.Out {
		sender = common.StyleSenderSelf.Render("You")
	} else if msg.Sender != "" {
		sender = common.StyleSender.Render(msg.Sender)
	} else {
		sender = common.StyleSender.Render("Unknown")
	}

	isMarked := m.selecting && m.selected[msg.ID]

	prefix := "  "
	if isSelected {
		if m.selecting {
			prefix = lipgloss.NewStyle().Foreground(common.ColorWarning).Render(">") + " "
		} else if m.inputFocused {
			prefix = lipgloss.NewStyle().Foreground(common.ColorMuted).Render(">") + " "
		} else {
			prefix = lipgloss.NewStyle().Foreground(common.ColorPrimary).Render(">") + " "
		}
	} else if isMarked {
		prefix = lipgloss.NewStyle().Foreground(common.ColorWarning).Render("*") + " "
	}

	if isExpanded && (msg.Text != "" || msg.Media != nil) {
		// Header line
		header := fmt.Sprintf("%s%s %s:", prefix, timestamp, sender)

		indent := "    "
		textWidth := m.width - len(indent)
		if textWidth < 20 {
			textWidth = 20
		}

		lines := []string{header}

		// Media label line
		if msg.Media != nil {
			lines = append(lines, indent+common.StyleMediaLabel.Render(msg.Media.Label))
		}

		// Photo thumbnail (or loading placeholder)
		if msg.Media != nil && msg.Media.Type == telegram.MediaPhoto {
			if rendered, ok := m.photoCache[msg.ID]; ok {
				for _, il := range strings.Split(rendered, "\n") {
					lines = append(lines, indent+il)
				}
			} else if m.photoLoading[msg.ID] {
				lines = append(lines, indent+common.StyleMuted.Render("[Loading photo...]"))
			}
		}

		// Full text, word-wrapped
		if msg.Text != "" {
			styledText := format.RenderStyledTextMultiline(msg.Text, msg.Entities, textWidth)
			for _, tl := range strings.Split(styledText, "\n") {
				lines = append(lines, indent+tl)
			}
		}

		// Reactions line (expanded)
		if len(msg.Reactions) > 0 {
			lines = append(lines, indent+renderReactions(msg.Reactions))
		}

		return lines
	}

	// Collapsed: single line
	var text string
	switch {
	case msg.Media != nil && msg.Text != "":
		text = common.StyleMediaLabel.Render(msg.Media.Label) + " " + format.RenderStyledText(msg.Text, msg.Entities)
	case msg.Media != nil:
		text = common.StyleMediaLabel.Render(msg.Media.Label)
	case msg.Text != "":
		text = format.RenderStyledText(msg.Text, msg.Entities)
	default:
		text = common.StyleMuted.Render("[empty message]")
	}

	// Append reactions to collapsed line
	reactions := ""
	if len(msg.Reactions) > 0 {
		reactions = " " + renderReactions(msg.Reactions)
	}

	line := fmt.Sprintf("%s%s %s: %s%s", prefix, timestamp, sender, text, reactions)
	return []string{lipgloss.NewStyle().MaxWidth(m.width).Render(line)}
}

func renderReactions(reactions []telegram.Reaction) string {
	reactionStyle := lipgloss.NewStyle().Foreground(common.ColorWarning)
	var parts []string
	for _, r := range reactions {
		parts = append(parts, reactionStyle.Render(fmt.Sprintf("%s%d", r.Emoji, r.Count)))
	}
	return strings.Join(parts, " ")
}

func (m Model) renderInput() string {
	style := lipgloss.NewStyle().
		MaxWidth(m.width).
		Padding(0, 1)

	prefix := common.StyleMuted.Render("> ")
	cursor := ""
	if m.focused && m.inputFocused {
		cursor = "█"
	}

	return style.Render(prefix + m.input + cursor)
}

func (m Model) maybeLoadOlder() (Model, tea.Cmd) {
	if m.loadingOlder || m.noMoreHistory || m.searchActive || len(m.messages) == 0 {
		return m, nil
	}
	if m.cursor == 0 {
		m.loadingOlder = true
		tg := m.tg
		chat := *m.chat
		offsetID := m.messages[0].ID
		return m, func() tea.Msg {
			return tg.FetchOlderHistory(chat, offsetID)()
		}
	}
	return m, nil
}

func (m Model) activeMessages() []telegram.Message {
	if m.searchActive && m.searchResults != nil {
		return m.searchResults
	}
	return m.messages
}

func (m Model) renderSearchInput() string {
	style := lipgloss.NewStyle().
		MaxWidth(m.width).
		Padding(0, 1)

	prefix := lipgloss.NewStyle().Foreground(common.ColorWarning).Render("/")
	cursor := "█"
	return style.Render(prefix + m.searchQuery + cursor)
}

// Helper methods

func (m Model) msgAreaHeight() int {
	inputHeight := 0
	if m.chat != nil && m.chat.Type != telegram.ChatTypeChannel {
		inputHeight = 1
	}
	searchHeight := 0
	if m.searching {
		searchHeight = 1
	}
	return m.height - 1 - inputHeight - searchHeight
}

func (m Model) visualHeight(msg telegram.Message) int {
	if msg.ID == m.expandedMsgID && (msg.Text != "" || msg.Media != nil) {
		h := 1 // header line
		if msg.Media != nil {
			h++ // media label line
		}
		// Photo thumbnail lines
		if msg.Media != nil && msg.Media.Type == telegram.MediaPhoto {
			if lines, ok := m.photoLines[msg.ID]; ok {
				h += lines
			} else if m.photoLoading[msg.ID] {
				h++ // "[Loading photo...]" placeholder
			}
		}
		if msg.Text != "" {
			indent := "    "
			textWidth := m.width - len(indent)
			if textWidth < 20 {
				textWidth = 20
			}
			styledText := format.RenderStyledTextMultiline(msg.Text, msg.Entities, textWidth)
			h += strings.Count(styledText, "\n") + 1
		}
		if len(msg.Reactions) > 0 {
			h++
		}
		return h
	}
	return 1
}

func (m *Model) ensureCursorVisible() {
	msgs := m.activeMessages()
	if len(msgs) == 0 || m.cursor < 0 {
		return
	}

	height := m.msgAreaHeight()
	if height <= 0 {
		return
	}

	// Calculate visual line positions
	linePos := 0
	msgTop := 0
	msgBottom := 0
	for i, msg := range msgs {
		h := m.visualHeight(msg)
		if i == m.cursor {
			msgTop = linePos
			msgBottom = linePos + h
		}
		linePos += h
	}
	totalLines := linePos

	// Current visible range
	visibleBottom := totalLines - m.scrollOffset
	visibleTop := visibleBottom - height

	// Adjust scroll to make cursor visible
	if msgTop < visibleTop {
		m.scrollOffset = totalLines - msgTop - height
	}
	if msgBottom > visibleBottom {
		m.scrollOffset = totalLines - msgBottom
	}

	// Clamp
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	maxScroll := totalLines - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}
}

func (m *Model) clampCursor() {
	msgs := m.activeMessages()
	if len(msgs) == 0 {
		m.cursor = -1
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(msgs) {
		m.cursor = len(msgs) - 1
	}
}

// Public accessors

func (m Model) SetChat(chat *telegram.Chat) Model {
	m.chat = chat
	m.messages = nil
	m.input = ""
	m.scrollOffset = 0
	m.cursor = -1
	m.expandedMsgID = -1
	m.inputFocused = chat.Type != telegram.ChatTypeChannel
	m.loadingOlder = false
	m.noMoreHistory = false
	m.photoCache = nil
	m.photoLines = nil
	m.photoLoading = nil
	m.fileSaving = nil
	m.selecting = false
	m.selected = nil
	m.searching = false
	m.searchQuery = ""
	m.searchResults = nil
	m.searchActive = false
	return m
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

func (m Model) Focused() bool {
	return m.focused
}

func (m Model) InputFocused() bool {
	return m.inputFocused
}

func (m Model) SetInputFocus(focused bool) Model {
	m.inputFocused = focused
	return m
}

func (m Model) IsSearching() bool {
	return m.searching
}

func (m Model) HasSearchResults() bool {
	return m.searchActive
}

func (m Model) IsSelecting() bool {
	return m.selecting
}

func (m Model) CancelSelection() Model {
	m.selecting = false
	m.selected = nil
	return m
}

func (m Model) HasExpanded() bool {
	return m.expandedMsgID != -1
}

func (m Model) CollapseExpanded() Model {
	m.expandedMsgID = -1
	return m
}

func (m *Model) initPhotoCaches() {
	if m.photoCache == nil {
		m.photoCache = make(map[int]string)
	}
	if m.photoLines == nil {
		m.photoLines = make(map[int]int)
	}
	if m.photoLoading == nil {
		m.photoLoading = make(map[int]bool)
	}
}

func (m Model) photoMaxWidth() int {
	w := m.width - 8 // indent + some margin
	if w > 40 {
		w = 40
	}
	if w < 10 {
		w = 10
	}
	return w
}

func (m Model) photoMaxHeight() int {
	h := m.msgAreaHeight() / 2
	if h > 15 {
		h = 15
	}
	if h < 5 {
		h = 5
	}
	return h
}

func (m Model) isDownloadable(info *telegram.MediaInfo) bool {
	switch info.Type {
	case telegram.MediaPhoto:
		return info.PhotoThumbSize != ""
	case telegram.MediaVideo, telegram.MediaDocument, telegram.MediaAudio, telegram.MediaVoice, telegram.MediaAnimation:
		return info.DocID != 0
	}
	return false
}

func (m Model) downloadPath(info *telegram.MediaInfo) string {
	dir := filepath.Join(os.Getenv("HOME"), "Downloads")
	os.MkdirAll(dir, 0o755)

	name := info.FileName
	if name == "" {
		ts := time.Now().Format("20060102_150405")
		switch info.Type {
		case telegram.MediaPhoto:
			name = "photo_" + ts + ".jpg"
		case telegram.MediaVideo:
			name = "video_" + ts + ".mp4"
		case telegram.MediaVoice:
			name = "voice_" + ts + ".ogg"
		case telegram.MediaAudio:
			name = "audio_" + ts + ".mp3"
		case telegram.MediaAnimation:
			name = "animation_" + ts + ".mp4"
		default:
			name = "file_" + ts
		}
	}

	path := filepath.Join(dir, name)
	// Avoid overwriting: add suffix if file exists
	if _, err := os.Stat(path); err == nil {
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		for i := 1; ; i++ {
			path = filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, i, ext))
			if _, err := os.Stat(path); os.IsNotExist(err) {
				break
			}
		}
	}
	return path
}
