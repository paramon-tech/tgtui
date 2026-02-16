package telegram

import (
	"sort"

	"github.com/gotd/td/tg"
)

type DialogsLoadedMsg struct {
	Chats []Chat
}

type DialogsErrorMsg struct {
	Err error
}

func (c *Client) FetchDialogs() func() interface{} {
	return func() interface{} {
		result, err := c.api.MessagesGetDialogs(c.ctx, &tg.MessagesGetDialogsRequest{
			OffsetPeer: &tg.InputPeerEmpty{},
			Limit:      100,
		})
		if err != nil {
			return DialogsErrorMsg{Err: err}
		}

		var chats []Chat

		switch r := result.(type) {
		case *tg.MessagesDialogs:
			chats = c.extractDialogs(r.Dialogs, r.Users, r.Chats, r.Messages)
		case *tg.MessagesDialogsSlice:
			chats = c.extractDialogs(r.Dialogs, r.Users, r.Chats, r.Messages)
		}

		// Partition: pinned chats first (preserve server order), then non-pinned by date
		var pinned, unpinned []Chat
		for _, c := range chats {
			if c.Pinned {
				pinned = append(pinned, c)
			} else {
				unpinned = append(unpinned, c)
			}
		}
		sort.Slice(unpinned, func(i, j int) bool {
			var di, dj int
			if unpinned[i].LastMessage != nil {
				di = unpinned[i].LastMessage.Date
			}
			if unpinned[j].LastMessage != nil {
				dj = unpinned[j].LastMessage.Date
			}
			return di > dj
		})
		chats = append(pinned, unpinned...)

		return DialogsLoadedMsg{Chats: chats}
	}
}

func (c *Client) extractDialogs(dialogs []tg.DialogClass, users []tg.UserClass, chatClasses []tg.ChatClass, messages []tg.MessageClass) []Chat {
	userMap := make(map[int64]*tg.User)
	for _, u := range users {
		if user, ok := u.(*tg.User); ok {
			userMap[user.ID] = user
		}
	}

	chatMap := make(map[int64]*tg.Chat)
	channelMap := make(map[int64]*tg.Channel)
	for _, ch := range chatClasses {
		switch v := ch.(type) {
		case *tg.Chat:
			chatMap[v.ID] = v
		case *tg.Channel:
			channelMap[v.ID] = v
		}
	}

	msgMap := make(map[int]tg.MessageClass)
	for _, m := range messages {
		switch v := m.(type) {
		case *tg.Message:
			msgMap[v.ID] = m
		case *tg.MessageService:
			msgMap[v.ID] = m
		}
	}

	var chats []Chat
	for _, d := range dialogs {
		dialog, ok := d.(*tg.Dialog)
		if !ok {
			continue
		}

		chat := Chat{
			UnreadCount: dialog.UnreadCount,
			Pinned:      dialog.Pinned,
		}

		switch peer := dialog.Peer.(type) {
		case *tg.PeerUser:
			user, exists := userMap[peer.UserID]
			if !exists {
				continue
			}
			chat.ID = user.ID
			chat.AccessHash = user.AccessHash
			chat.Title = displayName(user.FirstName, user.LastName)
			chat.Type = ChatTypePrivate

		case *tg.PeerChat:
			group, exists := chatMap[peer.ChatID]
			if !exists {
				continue
			}
			chat.ID = group.ID
			chat.Title = group.Title
			chat.Type = ChatTypeGroup

		case *tg.PeerChannel:
			channel, exists := channelMap[peer.ChannelID]
			if !exists {
				continue
			}
			chat.ID = channel.ID
			chat.AccessHash = channel.AccessHash
			chat.Title = channel.Title
			if channel.Broadcast {
				chat.Type = ChatTypeChannel
			} else {
				chat.Type = ChatTypeGroup
			}
		}

		if dialog.TopMessage != 0 {
			if m, exists := msgMap[dialog.TopMessage]; exists {
				if msg, ok := m.(*tg.Message); ok {
					sender := ""
					senderID := int64(0)
					if msg.FromID != nil {
						if p, ok := msg.FromID.(*tg.PeerUser); ok {
							senderID = p.UserID
							if u, exists := userMap[p.UserID]; exists {
								sender = displayName(u.FirstName, u.LastName)
							}
						}
					}
					chat.LastMessage = &Message{
						ID:       msg.ID,
						ChatID:   chat.ID,
						SenderID: senderID,
						Sender:   sender,
						Text:     msg.Message,
						Date:     msg.Date,
						Out:      msg.Out,
						Media:    extractMediaInfo(msg.Media),
					}
				}
			}
		}

		chats = append(chats, chat)
	}

	return chats
}
