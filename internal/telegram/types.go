package telegram

import "github.com/gotd/td/tg"

type ChatType int

const (
	ChatTypePrivate ChatType = iota
	ChatTypeGroup
	ChatTypeChannel
)

type Chat struct {
	ID          int64
	AccessHash  int64
	Title       string
	Type        ChatType
	UnreadCount int
	LastMessage *Message
}

type MediaType int

const (
	MediaPhoto MediaType = iota
	MediaVideo
	MediaDocument
	MediaAudio
	MediaVoice
	MediaSticker
	MediaAnimation
	MediaContact
	MediaLocation
	MediaPoll
	MediaOther
)

type MediaInfo struct {
	Type     MediaType
	Label    string // Pre-formatted: "[Photo]", "[Video 0:32]", etc.
	FileName string
	FileSize int64
	Width    int
	Height   int
	// Photo download fields (Phase 2)
	PhotoID         int64
	PhotoAccessHash int64
	PhotoFileRef    []byte
	PhotoDCID       int
	PhotoThumbSize  string
}

type Message struct {
	ID       int
	ChatID   int64
	SenderID int64
	Sender   string
	Text     string
	Date     int
	Out      bool
	Entities []tg.MessageEntityClass
	Media    *MediaInfo
}
