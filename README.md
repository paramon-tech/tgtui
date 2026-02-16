# tgtui

A lightweight Telegram TUI client for the terminal, focused on messaging.

Built with Go, [gotd](https://github.com/gotd/td) (pure Go MTProto), and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- Browse your Telegram chats with pinned chats shown first (matching mobile app order)
- Send and receive text messages in real time
- Rich text rendering: bold, italic, code, links, mentions, spoilers, and more
- Media support: descriptive labels for photos, videos, documents, stickers, voice messages, polls, contacts, and locations
- Photo thumbnails rendered directly in the terminal using half-block characters
- Download photos, videos, documents, and other media to disk with `D`
- Helix-inspired modal navigation (Normal/Insert modes)
- Supports private chats, groups, and channels (read-only)
- 3-step authentication: phone → code → 2FA password

## Requirements

- Go 1.25+
- Telegram API credentials (`API_ID` and `API_HASH` from [my.telegram.org](https://my.telegram.org))

## Install

```bash
go install github.com/paramon-tech/tgtui@latest
```

Or build from source:

```bash
git clone https://github.com/paramon-tech/tgtui.git
cd tgtui
go build -o tgtui .
```

## Usage

```bash
export TGTUI_API_ID=your_api_id
export TGTUI_API_HASH=your_api_hash
./tgtui
```

Session data is stored at `~/.local/share/tgtui/session.json`.

## Key Bindings

| Key | Chat List | Chat View (Normal) | Chat View (Insert) |
|-----|-----------|---------------------|---------------------|
| `Tab` | Switch to chat view | Switch to chat list | Switch to chat list |
| `Esc` | — | Collapse expanded msg | Exit to normal mode |
| `j/k` `↑/↓` | Navigate chats | Navigate messages | — |
| `Enter` | Open chat | Expand/collapse msg | Send message |
| `i` | — | Enter insert mode | — |
| `D` | — | Download media to ~/Downloads | — |
| `PgUp/PgDn` | — | Page scroll | Exit to normal + scroll |
| `Ctrl+C` | Quit | Quit | Quit |

## Media Support

Messages with media display descriptive labels instead of generic placeholders:

| Media Type | Display |
|---|---|
| Photo | `[Photo]` + inline thumbnail on expand |
| Video | `[Video 1:32]` |
| Document | `[Document: report.pdf (2.4 MB)]` |
| Voice | `[Voice 0:12]` |
| Audio | `[Audio: Song Title (3:45)]` |
| Sticker | `[Sticker 😀]` |
| GIF | `[GIF]` |
| Contact | `[Contact: John Doe]` |
| Location | `[Location]` / `[Live Location]` |
| Poll | `[Poll: What do you think?]` |

Press `Enter` on a photo message to see an inline thumbnail rendered with half-block characters. Press `D` on any media message to save it to `~/Downloads/`.

## License

GPLv3 — see [LICENSE](LICENSE).
