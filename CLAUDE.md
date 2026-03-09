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
- **rasterm** (`github.com/BourgeoisBear/rasterm`) — Kitty/iTerm2/Sixel image rendering
- **rsc.io/qr** — QR code generation for login
- API credentials: `TGTUI_API_ID` and `TGTUI_API_HASH` env vars
- Session stored at `~/.local/share/tgtui/session.json`

## Architecture

- `main.go` — Entry point: loads config, creates telegram client, starts TUI
- `internal/config/` — Env var loading, XDG data dir
- `internal/telegram/` — gotd client wrapper, auth, dialogs, messages, search, update dispatcher
- `internal/ui/` — Bubble Tea root model (`app.go`), screen routing, forward flow orchestration
- `internal/ui/common/` — Shared styles and message types (breaks import cycles)
- `internal/ui/auth/` — Auth screen: QR code or phone → code → 2FA password
- `internal/ui/chatlist/` — Left panel: scrollable chat list + forward destination picker
- `internal/ui/chatview/` — Right panel: messages viewport + text input + visual selection + search
- `internal/ui/statusbar/` — Bottom bar: mode indicator (NOR/INS/VIS/FWD/SRH) + status text
- `internal/format/` — Text entity rendering, multi-protocol image rendering, terminal detection

### Data Flow

- **gotd → TUI**: `telegram.Client` holds `*tea.Program`, calls `p.Send(msg)` from update handlers
- **TUI → gotd**: User actions return `tea.Cmd` functions that call `telegram.Client` methods
- **Sent messages**: Not optimistic — arrive back through the update dispatcher
- **History pagination**: Older messages loaded on-demand when scrolling to top via `FetchOlderHistory`

### Key Bindings

Helix-inspired modal navigation with Normal (NOR), Insert (INS), Visual (VIS), Forward (FWD), and Search (SRH) modes:

- `Tab` — switch focus between chat list and chat view
- `j/k` or `↑/↓` — navigate chat list / scroll messages
- `Enter` — select chat (in list) / expand message (in normal) / send (in insert)
- `i` — enter insert mode (chat view)
- `Esc` — exit insert mode / collapse expanded message / exit search results
- `v` — enter visual selection mode; `Space` — toggle selection; `f` — forward
- `/` — search messages in current chat; `n/N` — next/prev result
- `D` — download media to ~/Downloads
- `PgUp/PgDown` — page scroll messages (loads older history at top)
- `Ctrl+C` — quit
