package telegram

import (
	"context"
	"errors"
	"fmt"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

type AuthorizedMsg struct{}
type NeedAuthMsg struct{}

type CodeSentMsg struct {
	PhoneCodeHash string
}

type AuthErrorMsg struct {
	Err error
}

type Need2FAMsg struct{}

func (c *Client) SendCode(phone string) func() interface{} {
	return func() interface{} {
		sentCode, err := c.client.Auth().SendCode(c.ctx, phone, auth.SendCodeOptions{})
		if err != nil {
			return AuthErrorMsg{Err: err}
		}
		s, ok := sentCode.(*tg.AuthSentCode)
		if !ok {
			return AuthErrorMsg{Err: fmt.Errorf("unexpected sent code type: %T", sentCode)}
		}
		return CodeSentMsg{PhoneCodeHash: s.PhoneCodeHash}
	}
}

func (c *Client) SignIn(phone, code, phoneCodeHash string) func() interface{} {
	return func() interface{} {
		_, err := c.client.Auth().SignIn(c.ctx, phone, code, phoneCodeHash)
		if err != nil {
			if errors.Is(err, auth.ErrPasswordAuthNeeded) {
				return Need2FAMsg{}
			}
			return AuthErrorMsg{Err: err}
		}
		return c.afterAuth()
	}
}

func (c *Client) Submit2FA(password string) func() interface{} {
	return func() interface{} {
		_, err := c.client.Auth().Password(c.ctx, password)
		if err != nil {
			return AuthErrorMsg{Err: err}
		}
		return c.afterAuth()
	}
}

func (c *Client) afterAuth() interface{} {
	self, err := c.client.Self(context.Background())
	if err != nil {
		return AuthErrorMsg{Err: err}
	}
	c.selfID = self.ID
	return AuthorizedMsg{}
}
