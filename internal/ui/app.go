package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/paramon-tech/tgtui/internal/telegram"
	"github.com/paramon-tech/tgtui/internal/ui/auth"
	"github.com/paramon-tech/tgtui/internal/ui/chatlist"
	"github.com/paramon-tech/tgtui/internal/ui/chatview"
	"github.com/paramon-tech/tgtui/internal/ui/statusbar"
)

type screen int

const (
	screenLoading screen = iota
	screenAuth
	screenMain
)

type focusPane int

const (
	focusChatList focusPane = iota
	focusChatView
)

type App struct {
	tg            *telegram.Client
	screen        screen
	focus         focusPane
	auth          auth.Model
	chatList      chatlist.Model
	chatView      chatview.Model
	statusBar     statusbar.Model
	selectedChat  *telegram.Chat
	width, height int
	fatalErr      error
}

func NewApp(tg *telegram.Client) App {
	return App{
		tg:        tg,
		screen:    screenLoading,
		auth:      auth.New(tg),
		chatList:  chatlist.New(),
		chatView:  chatview.New(tg),
		statusBar: statusbar.New(),
	}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

		if a.screen == screenMain {
			if msg.String() == "tab" {
				a.toggleFocus()
				a.updateMode()
				return a, nil
			}
			if msg.String() == "esc" && a.focus == focusChatView {
				if a.chatView.InputFocused() {
					// Esc: exit input mode, stay in chat view
					a.chatView = a.chatView.SetInputFocus(false)
					a.updateMode()
					return a, nil
				}
				if a.chatView.HasExpanded() {
					// Esc with expanded message: collapse it
					a.chatView = a.chatView.CollapseExpanded()
					return a, nil
				}
			}
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateSizes()

	case NeedAuthMsg:
		a.screen = screenAuth

	case AuthorizedMsg:
		a.screen = screenMain
		a.statusBar, _ = a.statusBar.Update(msg)
		tg := a.tg
		return a, func() tea.Msg {
			return tg.FetchDialogs()()
		}

	case ChatSelectedMsg:
		chat := msg.Chat
		a.selectedChat = &chat
		a.chatView = a.chatView.SetChat(&chat)
		a.focus = focusChatView
		a.chatList = a.chatList.SetFocus(false)
		a.chatList = a.chatList.SetActiveChat(chat.ID)
		a.chatView = a.chatView.SetFocus(true)
		a.updateMode()
		tg := a.tg
		return a, func() tea.Msg {
			return tg.FetchHistory(chat)()
		}

	case FatalErrorMsg:
		a.fatalErr = msg.Err
		return a, tea.Quit
	}

	switch a.screen {
	case screenAuth:
		var cmd tea.Cmd
		a.auth, cmd = a.auth.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case screenMain:
		var cmd tea.Cmd

		a.chatList, cmd = a.chatList.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		a.chatView, cmd = a.chatView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		a.statusBar, cmd = a.statusBar.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		a.updateMode()
	}

	return a, tea.Batch(cmds...)
}

func (a App) View() string {
	if a.fatalErr != nil {
		return StyleError.Render("Fatal error: " + a.fatalErr.Error())
	}

	switch a.screen {
	case screenLoading:
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center,
			StyleMuted.Render("Connecting to Telegram..."))

	case screenAuth:
		return a.auth.View()

	case screenMain:
		return a.mainView()
	}

	return ""
}

func (a App) mainView() string {
	statusHeight := 1
	mainHeight := a.height - statusHeight

	listWidth := a.width * 30 / 100
	if listWidth < 20 {
		listWidth = 20
	}

	sepStr := ""
	for i := 0; i < mainHeight; i++ {
		if i > 0 {
			sepStr += "\n"
		}
		sepStr += "│"
	}
	separator := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render(sepStr)

	list := a.chatList.View()
	view := a.chatView.View()

	main := lipgloss.JoinHorizontal(lipgloss.Top, list, separator, view)
	status := a.statusBar.View()

	return lipgloss.JoinVertical(lipgloss.Left, main, status)
}

func (a *App) toggleFocus() {
	if a.focus == focusChatList {
		a.focus = focusChatView
		a.chatList = a.chatList.SetFocus(false)
		a.chatView = a.chatView.SetFocus(true)
	} else {
		a.focus = focusChatList
		a.chatList = a.chatList.SetFocus(true)
		a.chatView = a.chatView.SetFocus(false)
	}
}

func (a *App) currentMode() string {
	if a.focus == focusChatList {
		return "NOR"
	}
	if a.chatView.InputFocused() {
		return "INS"
	}
	return "NOR"
}

func (a *App) updateMode() {
	a.statusBar = a.statusBar.SetMode(a.currentMode())
}

func (a *App) updateSizes() {
	statusHeight := 1
	mainHeight := a.height - statusHeight

	listWidth := a.width * 30 / 100
	if listWidth < 20 {
		listWidth = 20
	}
	viewWidth := a.width - listWidth - 1

	a.auth = a.auth.SetSize(a.width, a.height)
	a.chatList = a.chatList.SetSize(listWidth, mainHeight)
	a.chatView = a.chatView.SetSize(viewWidth, mainHeight)
	a.statusBar = a.statusBar.SetSize(a.width)
}
