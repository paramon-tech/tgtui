package auth

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/paramon-tech/tgtui/internal/telegram"
	"github.com/paramon-tech/tgtui/internal/ui/common"
	"rsc.io/qr"
)

type step int

const (
	stepChooseMethod step = iota
	stepPhone
	stepCode
	stepPassword
	stepQR
)

type Model struct {
	tg            *telegram.Client
	step          step
	phone         string
	code          string
	password      string
	phoneCodeHash string
	codeType      string
	err           string
	loading       bool
	qrCode        string // rendered QR code for terminal
	methodCursor  int    // 0=QR, 1=Phone
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
		if m.loading && m.step != stepQR {
			return m, nil
		}

		if m.step == stepChooseMethod {
			return m.handleMethodChoice(msg)
		}

		if m.step == stepQR {
			// Only Esc goes back during QR
			if msg.String() == "esc" {
				m.step = stepChooseMethod
				m.qrCode = ""
				m.err = ""
				m.loading = false
			}
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

		if msg.String() == "esc" && m.step == stepPhone {
			m.step = stepChooseMethod
			m.err = ""
		}

	case common.CodeSentMsg:
		m.loading = false
		m.phoneCodeHash = msg.PhoneCodeHash
		m.codeType = msg.CodeType
		m.step = stepCode

	case common.Need2FAMsg:
		m.loading = false
		m.qrCode = ""
		m.step = stepPassword

	case common.AuthErrorMsg:
		m.loading = false
		m.err = msg.Err.Error()

	case common.QRTokenMsg:
		m.loading = false
		m.qrCode = renderQRCode(msg.URL)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m Model) handleMethodChoice(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.methodCursor > 0 {
			m.methodCursor--
		}
	case "down", "j":
		if m.methodCursor < 1 {
			m.methodCursor++
		}
	case "enter":
		if m.methodCursor == 0 {
			// QR code login
			m.step = stepQR
			m.loading = true
			m.err = ""
			tg := m.tg
			return m, func() tea.Msg {
				return tg.StartQRLogin(tg.LoggedIn())()
			}
		}
		m.step = stepPhone
		m.err = ""
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
	case stepChooseMethod:
		b.WriteString("Choose login method:\n\n")
		options := []string{"QR Code (scan with phone)", "Phone Number"}
		for i, opt := range options {
			if i == m.methodCursor {
				b.WriteString("  " + lipgloss.NewStyle().Foreground(common.ColorPrimary).Bold(true).Render("> "+opt) + "\n")
			} else {
				b.WriteString("    " + common.StyleMuted.Render(opt) + "\n")
			}
		}

	case stepQR:
		if m.qrCode != "" {
			b.WriteString("Scan with Telegram on your phone:\n\n")
			b.WriteString(m.qrCode)
			b.WriteString("\n")
			b.WriteString(common.StyleMuted.Render("  Open Telegram > Settings > Devices > Link Desktop Device"))
		} else {
			b.WriteString(common.StyleMuted.Render("  Generating QR code..."))
		}

	case stepPhone:
		b.WriteString("Enter your phone number (with country code):\n\n")
		b.WriteString("  Phone: ")
		b.WriteString(m.phone)
		b.WriteString("█")

	case stepCode:
		hint := "Enter the verification code"
		if m.codeType != "" {
			hint += " (sent via " + m.codeType + ")"
		}
		b.WriteString(hint + ":\n\n")
		b.WriteString("  Code: ")
		b.WriteString(m.code)
		b.WriteString("█")

	case stepPassword:
		b.WriteString("Enter your 2FA password:\n\n")
		b.WriteString("  Password: ")
		b.WriteString(strings.Repeat("•", len(m.password)))
		b.WriteString("█")
	}

	if m.loading && m.step != stepQR {
		b.WriteString("\n\n")
		b.WriteString(common.StyleMuted.Render("  Authenticating..."))
	}

	if m.err != "" {
		b.WriteString("\n\n")
		b.WriteString(common.StyleError.Render("  Error: " + m.err))
	}

	b.WriteString("\n\n")
	switch m.step {
	case stepChooseMethod:
		b.WriteString(common.StyleMuted.Render("  Press Enter to select • Ctrl+C to quit"))
	case stepQR:
		b.WriteString(common.StyleMuted.Render("  Press Esc to go back • Ctrl+C to quit"))
	default:
		b.WriteString(common.StyleMuted.Render("  Press Enter to submit • Esc to go back • Ctrl+C to quit"))
	}

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

// renderQRCode renders a QR code URL as Unicode half-block characters for the terminal.
func renderQRCode(url string) string {
	code, err := qr.Encode(url, qr.M)
	if err != nil {
		return "  [Failed to generate QR code]"
	}

	// Use upper half block (▀) technique:
	// Each terminal row represents 2 QR rows.
	// Background color = bottom pixel, foreground color = top pixel.
	// White = terminal default, Black = rendered block.
	size := code.Size
	quiet := 1 // quiet zone border

	var sb strings.Builder
	for y := -quiet; y < size+quiet; y += 2 {
		sb.WriteString("  ") // left padding
		for x := -quiet; x < size+quiet; x++ {
			topBlack := y >= 0 && y < size && x >= 0 && x < size && code.Black(x, y)
			bottomBlack := (y+1) >= 0 && (y+1) < size && x >= 0 && x < size && code.Black(x, y+1)

			switch {
			case topBlack && bottomBlack:
				sb.WriteString("██")
			case topBlack && !bottomBlack:
				sb.WriteString("▀▀")
			case !topBlack && bottomBlack:
				sb.WriteString("▄▄")
			default:
				sb.WriteString("  ")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
