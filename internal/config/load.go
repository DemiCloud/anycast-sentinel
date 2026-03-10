package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.AnycastIP == "" || cfg.AnycastDev == "" {
		return nil, fmt.Errorf("anycast_ip and anycast_dev are required")
	}
	return &cfg, nil
}
