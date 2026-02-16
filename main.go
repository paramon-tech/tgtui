package main

import (
	"fmt"
	"os"

	"github.com/paramon-tech/tgtui/internal/config"
	"github.com/paramon-tech/tgtui/internal/telegram"
	"github.com/paramon-tech/tgtui/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	tg := telegram.NewClient(cfg)

	p := tea.NewProgram(ui.NewApp(tg), tea.WithAltScreen())
	tg.SetProgram(p)

	go func() {
		if err := tg.Run(); err != nil {
			p.Send(ui.FatalErrorMsg{Err: err})
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	tg.Stop()
}
