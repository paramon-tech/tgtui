package telegram

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

type Message struct {
	ID       int
	ChatID   int64
	SenderID int64
	Sender   string
	Text     string
	Date     int
	Out      bool
}
