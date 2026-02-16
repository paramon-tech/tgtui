package telegram

import (
	"context"
	"sync"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/tg"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/paramon-tech/tgtui/internal/config"
)

type Client struct {
	cfg    *config.Config
	client *telegram.Client
	api    *tg.Client

	p  *tea.Program
	mu sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
	selfID int64
}

func NewClient(cfg *config.Config) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *Client) SetProgram(p *tea.Program) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.p = p
}

func (c *Client) send(msg tea.Msg) {
	c.mu.Lock()
	p := c.p
	c.mu.Unlock()
	if p != nil {
		p.Send(msg)
	}
}

func (c *Client) SelfID() int64 {
	return c.selfID
}

func (c *Client) Run() error {
	dispatcher := tg.NewUpdateDispatcher()
	c.setupHandlers(dispatcher)

	c.client = telegram.NewClient(c.cfg.APIId, c.cfg.APIHash, telegram.Options{
		SessionStorage: &FileSessionStorage{Path: c.cfg.SessionPath()},
		UpdateHandler:  dispatcher,
		DCList:         dcs.Prod(),
	})

	return c.client.Run(c.ctx, func(ctx context.Context) error {
		c.api = c.client.API()

		auth, err := c.client.Auth().Status(ctx)
		if err != nil {
			return err
		}

		if auth.Authorized {
			self, err := c.client.Self(ctx)
			if err != nil {
				return err
			}
			c.selfID = self.ID
			c.send(AuthorizedMsg{})
		} else {
			c.send(NeedAuthMsg{})
		}

		<-ctx.Done()
		return ctx.Err()
	})
}

func (c *Client) Stop() {
	c.cancel()
}

func (c *Client) API() *tg.Client {
	return c.api
}

func (c *Client) Context() context.Context {
	return c.ctx
}
