# tgtui

Telegram TUI client focused on text messaging only.

## Features

- Browse and search your Telegram chats
- Send and receive text messages in real time
- Helix-inspired modal navigation (Normal/Insert modes)
- Active chat indicator in the chat list
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
| `PgUp/PgDn` | — | Page scroll | Exit to normal + scroll |
| `Ctrl+C` | Quit | Quit | Quit |

## License

GPLv3 — see [LICENSE](LICENSE).
