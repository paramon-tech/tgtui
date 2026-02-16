package ui

import "github.com/paramon-tech/tgtui/internal/ui/common"

// Re-export message types used by app.go.
type (
	AuthorizedMsg       = common.AuthorizedMsg
	NeedAuthMsg         = common.NeedAuthMsg
	DialogsLoadedMsg    = common.DialogsLoadedMsg
	HistoryLoadedMsg    = common.HistoryLoadedMsg
	NewMessageMsg       = common.NewMessageMsg
	ChatSelectedMsg     = common.ChatSelectedMsg
	FatalErrorMsg       = common.FatalErrorMsg
	StatusMsg           = common.StatusMsg
	MessageSendErrorMsg = common.MessageSendErrorMsg
)
