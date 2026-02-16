package auth

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/paramon-tech/tgtui/internal/telegram"
	"github.com/paramon-tech/tgtui/internal/ui/common"
)

type step int

const (
	stepPhone step = iota
	stepCode
	stepPassword
)

type Model struct {
	tg            *telegram.Client
	step          step
	phone         string
	code          string
	password      string
	phoneCodeHash string
	err           string
	loading       bool
	width, height int
}

func New(tg *telegram.Client) Model {
	return Model{tg: tg}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.Type {
		case tea.KeyEnter:
			return m.submit()
		case tea.KeyBackspace:
			m.err = ""
			switch m.step {
			case stepPhone:
				if len(m.phone) > 0 {
					m.phone = m.phone[:len(m.phone)-1]
				}
			case stepCode:
				if len(m.code) > 0 {
					m.code = m.code[:len(m.code)-1]
				}
			case stepPassword:
				if len(m.password) > 0 {
					m.password = m.password[:len(m.password)-1]
				}
			}
		case tea.KeyRunes:
			r := string(msg.Runes)
			m.err = ""
			switch m.step {
			case stepPhone:
				m.phone += r
			case stepCode:
				m.code += r
			case stepPassword:
				m.password += r
			}
		}

	case common.CodeSentMsg:
		m.loading = false
		m.phoneCodeHash = msg.PhoneCodeHash
		m.step = stepCode

	case common.Need2FAMsg:
		m.loading = false
		m.step = stepPassword

	case common.AuthErrorMsg:
		m.loading = false
		m.err = msg.Err.Error()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m Model) submit() (Model, tea.Cmd) {
	switch m.step {
	case stepPhone:
		phone := strings.TrimSpace(m.phone)
		if phone == "" {
			m.err = "Phone number is required"
			return m, nil
		}
		m.loading = true
		m.err = ""
		return m, func() tea.Msg {
			return m.tg.SendCode(phone)()
		}

	case stepCode:
		code := strings.TrimSpace(m.code)
		if code == "" {
			m.err = "Code is required"
			return m, nil
		}
		m.loading = true
		m.err = ""
		phone := strings.TrimSpace(m.phone)
		hash := m.phoneCodeHash
		return m, func() tea.Msg {
			return m.tg.SignIn(phone, code, hash)()
		}

	case stepPassword:
		password := m.password
		if password == "" {
			m.err = "Password is required"
			return m, nil
		}
		m.loading = true
		m.err = ""
		return m, func() tea.Msg {
			return m.tg.Submit2FA(password)()
		}
	}

	return m, nil
}

func (m Model) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(common.ColorPrimary).
		MarginBottom(1)

	var b strings.Builder

	b.WriteString(titleStyle.Render("tgtui — Telegram TUI"))
	b.WriteString("\n\n")

	switch m.step {
	case stepPhone:
		b.WriteString("Enter your phone number (with country code):\n\n")
		b.WriteString("  Phone: ")
		b.WriteString(m.phone)
		b.WriteString("█")

	case stepCode:
		b.WriteString("Enter the verification code:\n\n")
		b.WriteString("  Code: ")
		b.WriteString(m.code)
		b.WriteString("█")

	case stepPassword:
		b.WriteString("Enter your 2FA password:\n\n")
		b.WriteString("  Password: ")
		b.WriteString(strings.Repeat("•", len(m.password)))
		b.WriteString("█")
	}

	if m.loading {
		b.WriteString("\n\n")
		b.WriteString(common.StyleMuted.Render("  Authenticating..."))
	}

	if m.err != "" {
		b.WriteString("\n\n")
		b.WriteString(common.StyleError.Render("  Error: " + m.err))
	}

	b.WriteString("\n\n")
	b.WriteString(common.StyleMuted.Render("  Press Enter to submit • Ctrl+C to quit"))

	content := b.String()

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	return m
}
