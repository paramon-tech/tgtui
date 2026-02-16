package telegram

import (
	"context"
	"encoding/json"
	"os"

	"github.com/gotd/td/session"
)

type FileSessionStorage struct {
	Path string
}

func (s *FileSessionStorage) LoadSession(_ context.Context) ([]byte, error) {
	data, err := os.ReadFile(s.Path)
	if os.IsNotExist(err) {
		return nil, session.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var stored struct {
		Data []byte `json:"data"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}
	return stored.Data, nil
}

func (s *FileSessionStorage) StoreSession(_ context.Context, data []byte) error {
	stored := struct {
		Data []byte `json:"data"`
	}{Data: data}

	out, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	return os.WriteFile(s.Path, out, 0600)
}
