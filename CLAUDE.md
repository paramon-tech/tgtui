# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

tgtui is a Telegram TUI (Terminal User Interface) client focused on text messaging only. Licensed under GPLv3.

## Build & Run

```bash
go build ./...           # Build
go vet ./...             # Lint
TGTUI_API_ID=... TGTUI_API_HASH=... ./tgtui  # Run
```

## Tech Stack

- **Go** with `github.com/gotd/td` (pure Go MTProto client)
- **Bubble Tea** v1 (`github.com/charmbracelet/bubbletea`) + Lipgloss v1
- API credentials: `TGTUI_API_ID` and `TGTUI_API_HASH` env vars
- Session stored at `~/.local/share/tgtui/session.json`

## Architecture

- `main.go` — Entry point: loads config, creates telegram client, starts TUI
- `internal/config/` — Env var loading, XDG data dir
- `internal/telegram/` — gotd client wrapper, auth, dialogs, messages, update dispatcher
- `internal/ui/` — Bubble Tea root model (`app.go`), screen routing
- `internal/ui/common/` — Shared styles and message types (breaks import cycles)
- `internal/ui/auth/` — 3-step auth screen (phone → code → 2FA password)
- `internal/ui/chatlist/` — Left panel: scrollable chat list
- `internal/ui/chatview/` — Right panel: messages viewport + text input
- `internal/ui/statusbar/` — Bottom bar: connection status

### Data Flow

- **gotd → TUI**: `telegram.Client` holds `*tea.Program`, calls `p.Send(msg)` from update handlers
- **TUI → gotd**: User actions return `tea.Cmd` functions that call `telegram.Client` methods
- **Sent messages**: Not optimistic — arrive back through the update dispatcher

### Key Bindings

- `Tab` — switch focus between chat list and chat view
- `j/k` or `↑/↓` — navigate chat list / scroll messages
- `Enter` — select chat (in list) / send message (in input)
- `Esc` — back to chat list from chat view
- `PgUp/PgDown` — scroll messages
- `Ctrl+C` — quit
