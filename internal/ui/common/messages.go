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
	OlderHistoryLoadedMsg = telegram.OlderHistoryLoadedMsg
	OlderHistoryErrorMsg  = telegram.OlderHistoryErrorMsg
	NewMessageMsg         = telegram.NewMessageMsg
	MessageSentMsg        = telegram.MessageSentMsg
	MessageSendErrorMsg   = telegram.MessageSendErrorMsg
	DownloadPhotoMsg      = telegram.DownloadPhotoMsg
	DownloadPhotoErrorMsg = telegram.DownloadPhotoErrorMsg
	SaveFileMsg           = telegram.SaveFileMsg
	SaveFileErrorMsg      = telegram.SaveFileErrorMsg
	ForwardedMsg          = telegram.ForwardedMsg
	ForwardErrorMsg       = telegram.ForwardErrorMsg
	ReactionsUpdatedMsg   = telegram.ReactionsUpdatedMsg
	QRTokenMsg            = telegram.QRTokenMsg
	SearchResultMsg       = telegram.SearchResultMsg
	SearchErrorMsg        = telegram.SearchErrorMsg
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

// ForwardRequestMsg is sent when the user has selected messages and pressed 'f'.
type ForwardRequestMsg struct {
	FromChat   telegram.Chat
	MessageIDs []int
}

// ForwardDestSelectedMsg is sent when the user picks a destination chat for forwarding.
type ForwardDestSelectedMsg struct {
	Chat telegram.Chat
}
