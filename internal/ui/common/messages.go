package common

import "github.com/paramon-tech/tgtui/internal/telegram"

// Re-export telegram messages as UI messages for convenience.
type (
	AuthorizedMsg         = telegram.AuthorizedMsg
	NeedAuthMsg           = telegram.NeedAuthMsg
	CodeSentMsg           = telegram.CodeSentMsg
	AuthErrorMsg          = telegram.AuthErrorMsg
	Need2FAMsg            = telegram.Need2FAMsg
	DialogsLoadedMsg      = telegram.DialogsLoadedMsg
	DialogsErrorMsg       = telegram.DialogsErrorMsg
	HistoryLoadedMsg      = telegram.HistoryLoadedMsg
	HistoryErrorMsg       = telegram.HistoryErrorMsg
	NewMessageMsg         = telegram.NewMessageMsg
	MessageSentMsg        = telegram.MessageSentMsg
	MessageSendErrorMsg   = telegram.MessageSendErrorMsg
	DownloadPhotoMsg      = telegram.DownloadPhotoMsg
	DownloadPhotoErrorMsg = telegram.DownloadPhotoErrorMsg
)

// FatalErrorMsg is sent when the telegram client encounters a fatal error.
type FatalErrorMsg struct {
	Err error
}

// ChatSelectedMsg is sent when a user selects a chat from the list.
type ChatSelectedMsg struct {
	Chat telegram.Chat
}

// StatusMsg updates the status bar text.
type StatusMsg struct {
	Text string
}
