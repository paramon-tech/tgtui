package telegram

import (
	"context"
	"errors"
	"fmt"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tgerr"
	"github.com/gotd/td/tg"
)

type AuthorizedMsg struct{}
type NeedAuthMsg struct{}

type CodeSentMsg struct {
	PhoneCodeHash string
	CodeType      string // hint about where code was sent
}

type AuthErrorMsg struct {
	Err error
}

type Need2FAMsg struct{}

type QRTokenMsg struct {
	URL string // tg://login?token=... to encode as QR
}

type QRAuthDoneMsg struct{}

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
		return CodeSentMsg{
			PhoneCodeHash: s.PhoneCodeHash,
			CodeType:      describeCodeType(s.Type),
		}
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

func (c *Client) StartQRLogin(loggedIn qrlogin.LoggedIn) func() interface{} {
	return func() interface{} {
		qr := qrlogin.NewQR(c.api, c.cfg.APIId, c.cfg.APIHash, qrlogin.Options{
			Migrate: c.client.MigrateTo,
		})

		_, err := qr.Auth(c.ctx, loggedIn, func(ctx context.Context, token qrlogin.Token) error {
			c.send(QRTokenMsg{URL: token.URL()})
			return nil
		})
		if err != nil {
			if tgerr.Is(err, "SESSION_PASSWORD_NEEDED") {
				return Need2FAMsg{}
			}
			return AuthErrorMsg{Err: err}
		}

		return c.afterAuth()
	}
}

func describeCodeType(t tg.AuthSentCodeTypeClass) string {
	switch t.(type) {
	case *tg.AuthSentCodeTypeApp:
		return "Telegram app"
	case *tg.AuthSentCodeTypeSMS:
		return "SMS"
	case *tg.AuthSentCodeTypeCall:
		return "phone call"
	case *tg.AuthSentCodeTypeFlashCall:
		return "flash call"
	case *tg.AuthSentCodeTypeMissedCall:
		return "missed call"
	case *tg.AuthSentCodeTypeEmailCode:
		return "email"
	case *tg.AuthSentCodeTypeFragmentSMS:
		return "Fragment SMS"
	case *tg.AuthSentCodeTypeFirebaseSMS:
		return "Firebase SMS"
	case *tg.AuthSentCodeTypeSMSWord:
		return "SMS (word)"
	case *tg.AuthSentCodeTypeSMSPhrase:
		return "SMS (phrase)"
	default:
		return "Telegram"
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
