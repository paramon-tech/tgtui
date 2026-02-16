package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	APIId   int
	APIHash string
	DataDir string
}

func Load() (*Config, error) {
	apiIDStr := os.Getenv("TGTUI_API_ID")
	if apiIDStr == "" {
		return nil, fmt.Errorf("TGTUI_API_ID environment variable is required")
	}
	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		return nil, fmt.Errorf("TGTUI_API_ID must be an integer: %w", err)
	}

	apiHash := os.Getenv("TGTUI_API_HASH")
	if apiHash == "" {
		return nil, fmt.Errorf("TGTUI_API_HASH environment variable is required")
	}

	dataDir, err := dataDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to determine data directory: %w", err)
	}

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &Config{
		APIId:   apiID,
		APIHash: apiHash,
		DataDir: dataDir,
	}, nil
}

func (c *Config) SessionPath() string {
	return filepath.Join(c.DataDir, "session.json")
}

func dataDirectory() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "tgtui"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "tgtui"), nil
}
