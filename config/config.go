package config

import (
	"encoding/json"
	"errors"
	"os"
)

type Config struct {
	X       int     `json:"x"`
	Y       int     `json:"y"`
	Locked  bool    `json:"locked"` // 固定位置（锁定后不可拖动）
	Visible bool    `json:"visible"`
	Scale   float64 `json:"scale"` // 整体缩放（1.0=默认大小，拖角放大，最小 1.0）
}

// Default returns the out-of-box config. X=-1 means "center screen at runtime".
func Default() Config {
	return Config{X: -1, Y: 16, Locked: false, Visible: true, Scale: 1.0}
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
	if cfg.Scale < 1.0 {
		cfg.Scale = 1.0 // 旧 config（无 scale 字段→0）或非法值兜底
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(path, data, 0644)
}
