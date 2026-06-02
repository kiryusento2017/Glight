package config

import (
	"encoding/json"
	"errors"
	"os"
)

type Config struct {
	X            int  `json:"x"`
	Y            int  `json:"y"`
	ClickThrough bool `json:"click_through"`
	Visible      bool `json:"visible"`
}

// Default returns the out-of-box config. X=-1 means "center screen at runtime".
func Default() Config {
	return Config{X: -1, Y: 16, ClickThrough: false, Visible: true}
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), nil // corrupt → safe fallback
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(path, data, 0644)
}
