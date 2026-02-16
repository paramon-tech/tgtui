package telegram

import (
	"context"

	"github.com/gotd/td/tg"
)

type NewMessageMsg struct {
	Message Message
}

func (c *Client) setupHandlers(dispatcher tg.UpdateDispatcher) {
	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		msg, ok := update.Message.(*tg.Message)
		if !ok {
			return nil
		}

		sender := ""
		senderID := int64(0)
		if msg.FromID != nil {
			if peer, ok := msg.FromID.(*tg.PeerUser); ok {
				senderID = peer.UserID
				if u, exists := e.Users[peer.UserID]; exists {
					sender = displayName(u.FirstName, u.LastName)
				}
			}
		}

		chatID := extractChatID(msg.PeerID)

		c.send(NewMessageMsg{
			Message: Message{
				ID:       msg.ID,
				ChatID:   chatID,
				SenderID: senderID,
				Sender:   sender,
				Text:     msg.Message,
				Date:     msg.Date,
				Out:      msg.Out,
				Entities: msg.Entities,
				Media:    extractMediaInfo(msg.Media),
			},
		})
		return nil
	})

	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
		msg, ok := update.Message.(*tg.Message)
		if !ok {
			return nil
		}

		sender := ""
		senderID := int64(0)
		if msg.FromID != nil {
			if peer, ok := msg.FromID.(*tg.PeerUser); ok {
				senderID = peer.UserID
				if u, exists := e.Users[peer.UserID]; exists {
					sender = displayName(u.FirstName, u.LastName)
				}
			}
		}

		chatID := extractChatID(msg.PeerID)

		c.send(NewMessageMsg{
			Message: Message{
				ID:       msg.ID,
				ChatID:   chatID,
				SenderID: senderID,
				Sender:   sender,
				Text:     msg.Message,
				Date:     msg.Date,
				Out:      msg.Out,
				Entities: msg.Entities,
				Media:    extractMediaInfo(msg.Media),
			},
		})
		return nil
	})
}

func extractChatID(peer tg.PeerClass) int64 {
	switch p := peer.(type) {
	case *tg.PeerUser:
		return p.UserID
	case *tg.PeerChat:
		return p.ChatID
	case *tg.PeerChannel:
		return p.ChannelID
	default:
		return 0
	}
}

func displayName(first, last string) string {
	if last == "" {
		return first
	}
	return first + " " + last
}
