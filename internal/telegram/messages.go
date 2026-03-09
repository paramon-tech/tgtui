package telegram

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/gotd/td/tg"
)

type HistoryLoadedMsg struct {
	ChatID   int64
	Messages []Message
}

type HistoryErrorMsg struct {
	Err error
}

type OlderHistoryLoadedMsg struct {
	ChatID   int64
	Messages []Message
}

type OlderHistoryErrorMsg struct {
	Err error
}

type MessageSentMsg struct {
	ChatID int64
}

type MessageSendErrorMsg struct {
	Err error
}

type ForwardedMsg struct {
	FromChatID int64
	ToChatID   int64
	Count      int
}

type ForwardErrorMsg struct {
	Err error
}

func (c *Client) FetchHistory(chat Chat) func() interface{} {
	return func() interface{} {
		peer := c.chatToInputPeer(chat)

		result, err := c.api.MessagesGetHistory(c.ctx, &tg.MessagesGetHistoryRequest{
			Peer:  peer,
			Limit: 50,
		})
		if err != nil {
			return HistoryErrorMsg{Err: err}
		}

		var msgs []Message

		var tgMessages []tg.MessageClass
		var users []tg.UserClass

		switch r := result.(type) {
		case *tg.MessagesMessages:
			tgMessages = r.Messages
			users = r.Users
		case *tg.MessagesMessagesSlice:
			tgMessages = r.Messages
			users = r.Users
		case *tg.MessagesChannelMessages:
			tgMessages = r.Messages
			users = r.Users
		}

		userMap := make(map[int64]*tg.User)
		for _, u := range users {
			if user, ok := u.(*tg.User); ok {
				userMap[user.ID] = user
			}
		}

		for _, m := range tgMessages {
			msg, ok := m.(*tg.Message)
			if !ok {
				continue
			}

			sender := ""
			senderID := int64(0)
			if msg.FromID != nil {
				if peer, ok := msg.FromID.(*tg.PeerUser); ok {
					senderID = peer.UserID
					if u, exists := userMap[peer.UserID]; exists {
						sender = displayName(u.FirstName, u.LastName)
					}
				}
			}

			msgs = append(msgs, Message{
				ID:        msg.ID,
				ChatID:    chat.ID,
				SenderID:  senderID,
				Sender:    sender,
				Text:      msg.Message,
				Date:      msg.Date,
				Out:       msg.Out,
				Entities:  msg.Entities,
				Media:     extractMediaInfo(msg.Media),
				Reactions: extractReactions(msg.Reactions),
			})
		}

		// Reverse to chronological order
		for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
			msgs[i], msgs[j] = msgs[j], msgs[i]
		}

		return HistoryLoadedMsg{ChatID: chat.ID, Messages: msgs}
	}
}

func (c *Client) FetchOlderHistory(chat Chat, offsetID int) func() interface{} {
	return func() interface{} {
		peer := c.chatToInputPeer(chat)

		result, err := c.api.MessagesGetHistory(c.ctx, &tg.MessagesGetHistoryRequest{
			Peer:     peer,
			OffsetID: offsetID,
			Limit:    50,
		})
		if err != nil {
			return OlderHistoryErrorMsg{Err: err}
		}

		var msgs []Message

		var tgMessages []tg.MessageClass
		var users []tg.UserClass

		switch r := result.(type) {
		case *tg.MessagesMessages:
			tgMessages = r.Messages
			users = r.Users
		case *tg.MessagesMessagesSlice:
			tgMessages = r.Messages
			users = r.Users
		case *tg.MessagesChannelMessages:
			tgMessages = r.Messages
			users = r.Users
		}

		userMap := make(map[int64]*tg.User)
		for _, u := range users {
			if user, ok := u.(*tg.User); ok {
				userMap[user.ID] = user
			}
		}

		for _, m := range tgMessages {
			msg, ok := m.(*tg.Message)
			if !ok {
				continue
			}

			sender := ""
			senderID := int64(0)
			if msg.FromID != nil {
				if peer, ok := msg.FromID.(*tg.PeerUser); ok {
					senderID = peer.UserID
					if u, exists := userMap[peer.UserID]; exists {
						sender = displayName(u.FirstName, u.LastName)
					}
				}
			}

			msgs = append(msgs, Message{
				ID:        msg.ID,
				ChatID:    chat.ID,
				SenderID:  senderID,
				Sender:    sender,
				Text:      msg.Message,
				Date:      msg.Date,
				Out:       msg.Out,
				Entities:  msg.Entities,
				Media:     extractMediaInfo(msg.Media),
				Reactions: extractReactions(msg.Reactions),
			})
		}

		// Reverse to chronological order
		for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
			msgs[i], msgs[j] = msgs[j], msgs[i]
		}

		return OlderHistoryLoadedMsg{ChatID: chat.ID, Messages: msgs}
	}
}

func (c *Client) SendMessage(chat Chat, text string) func() interface{} {
	return func() interface{} {
		peer := c.chatToInputPeer(chat)

		_, err := c.api.MessagesSendMessage(c.ctx, &tg.MessagesSendMessageRequest{
			Peer:     peer,
			Message:  text,
			RandomID: randomID(),
		})
		if err != nil {
			return MessageSendErrorMsg{Err: err}
		}

		return MessageSentMsg{ChatID: chat.ID}
	}
}

func (c *Client) ForwardMessages(fromChat Chat, messageIDs []int, toChat Chat) func() interface{} {
	return func() interface{} {
		fromPeer := c.chatToInputPeer(fromChat)
		toPeer := c.chatToInputPeer(toChat)

		randomIDs := make([]int64, len(messageIDs))
		for i := range randomIDs {
			randomIDs[i] = randomID()
		}

		_, err := c.api.MessagesForwardMessages(c.ctx, &tg.MessagesForwardMessagesRequest{
			FromPeer: fromPeer,
			ID:       messageIDs,
			RandomID: randomIDs,
			ToPeer:   toPeer,
		})
		if err != nil {
			return ForwardErrorMsg{Err: err}
		}

		return ForwardedMsg{
			FromChatID: fromChat.ID,
			ToChatID:   toChat.ID,
			Count:      len(messageIDs),
		}
	}
}

func (c *Client) chatToInputPeer(chat Chat) tg.InputPeerClass {
	switch chat.Type {
	case ChatTypePrivate:
		return &tg.InputPeerUser{UserID: chat.ID, AccessHash: chat.AccessHash}
	case ChatTypeGroup:
		if chat.AccessHash != 0 {
			return &tg.InputPeerChannel{ChannelID: chat.ID, AccessHash: chat.AccessHash}
		}
		return &tg.InputPeerChat{ChatID: chat.ID}
	case ChatTypeChannel:
		return &tg.InputPeerChannel{ChannelID: chat.ID, AccessHash: chat.AccessHash}
	default:
		return &tg.InputPeerEmpty{}
	}
}

func extractReactions(reactions tg.MessageReactions) []Reaction {
	if len(reactions.Results) == 0 {
		return nil
	}
	result := make([]Reaction, 0, len(reactions.Results))
	for _, r := range reactions.Results {
		emoji := ""
		switch re := r.Reaction.(type) {
		case *tg.ReactionEmoji:
			emoji = re.Emoticon
		case *tg.ReactionCustomEmoji:
			emoji = "✨" // custom emoji placeholder
		case *tg.ReactionPaid:
			emoji = "⭐"
		default:
			continue
		}
		_, chosen := r.GetChosenOrder()
		result = append(result, Reaction{
			Emoji:  emoji,
			Count:  r.Count,
			Chosen: chosen,
		})
	}
	return result
}

type SearchResultMsg struct {
	ChatID   int64
	Query    string
	Messages []Message
}

type SearchErrorMsg struct {
	Err error
}

func (c *Client) SearchHistory(chat Chat, query string) func() interface{} {
	return func() interface{} {
		peer := c.chatToInputPeer(chat)

		result, err := c.api.MessagesSearch(c.ctx, &tg.MessagesSearchRequest{
			Peer:   peer,
			Q:      query,
			Filter: &tg.InputMessagesFilterEmpty{},
			Limit:  50,
		})
		if err != nil {
			return SearchErrorMsg{Err: err}
		}

		var msgs []Message

		var tgMessages []tg.MessageClass
		var users []tg.UserClass

		switch r := result.(type) {
		case *tg.MessagesMessages:
			tgMessages = r.Messages
			users = r.Users
		case *tg.MessagesMessagesSlice:
			tgMessages = r.Messages
			users = r.Users
		case *tg.MessagesChannelMessages:
			tgMessages = r.Messages
			users = r.Users
		}

		userMap := make(map[int64]*tg.User)
		for _, u := range users {
			if user, ok := u.(*tg.User); ok {
				userMap[user.ID] = user
			}
		}

		for _, m := range tgMessages {
			msg, ok := m.(*tg.Message)
			if !ok {
				continue
			}

			sender := ""
			senderID := int64(0)
			if msg.FromID != nil {
				if peer, ok := msg.FromID.(*tg.PeerUser); ok {
					senderID = peer.UserID
					if u, exists := userMap[peer.UserID]; exists {
						sender = displayName(u.FirstName, u.LastName)
					}
				}
			}

			msgs = append(msgs, Message{
				ID:        msg.ID,
				ChatID:    chat.ID,
				SenderID:  senderID,
				Sender:    sender,
				Text:      msg.Message,
				Date:      msg.Date,
				Out:       msg.Out,
				Entities:  msg.Entities,
				Media:     extractMediaInfo(msg.Media),
				Reactions: extractReactions(msg.Reactions),
			})
		}

		// Reverse to chronological order
		for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
			msgs[i], msgs[j] = msgs[j], msgs[i]
		}

		return SearchResultMsg{ChatID: chat.ID, Query: query, Messages: msgs}
	}
}

func randomID() int64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int64(binary.LittleEndian.Uint64(b[:]))
}
