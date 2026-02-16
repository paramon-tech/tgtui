package common

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary   = lipgloss.Color("#7AA2F7")
	ColorSecondary = lipgloss.Color("#9ECE6A")
	ColorMuted     = lipgloss.Color("#565F89")
	ColorError     = lipgloss.Color("#F7768E")

	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	StyleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	StyleMuted = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StyleError = lipgloss.NewStyle().
			Foreground(ColorError)

	StyleSender = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	StyleSenderSelf = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	StyleTimestamp = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StyleUnread = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#E0AF68"))
)
