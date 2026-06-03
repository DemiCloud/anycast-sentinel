package config

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
)

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	dec := toml.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}

	if cfg.General.Dev == "" {
		return nil, fmt.Errorf("general.dev is required")
	}
	if cfg.General.Timeout != "" {
		d, err := time.ParseDuration(cfg.General.Timeout)
		if err != nil || d <= 0 {
			return nil, fmt.Errorf("general.timeout: invalid duration %q (e.g. \"30s\", \"2m\")", cfg.General.Timeout)
		}
	}
	if cfg.General.IP4 == "" && cfg.General.IP6 == "" {
		return nil, fmt.Errorf("at least one of general.ip4 or general.ip6 is required")
	}
	if cfg.General.IP4 != "" {
		parsed := net.ParseIP(cfg.General.IP4)
		if parsed == nil || parsed.To4() == nil {
			return nil, fmt.Errorf("general.ip4: invalid IPv4 address %q", cfg.General.IP4)
		}
	}
	if cfg.General.IP6 != "" {
		parsed := net.ParseIP(cfg.General.IP6)
		if parsed == nil || parsed.To4() != nil {
			return nil, fmt.Errorf("general.ip6: invalid IPv6 address %q", cfg.General.IP6)
		}
	}

	if len(cfg.Checks) == 0 {
		return nil, fmt.Errorf("at least one check must be defined")
	}
	seen := make(map[string]bool)
	for i, c := range cfg.Checks {
		if c.Name == "" {
			return nil, fmt.Errorf("check[%d]: name is required", i)
		}
		if seen[c.Name] {
			return nil, fmt.Errorf("check[%d]: duplicate name %q", i, c.Name)
		}
		seen[c.Name] = true
		switch c.Type {
		case HealthSystemd:
			if c.Unit == "" {
				return nil, fmt.Errorf("check %q: unit is required for systemd checks", c.Name)
			}
			if c.Timeout != "" {
				return nil, fmt.Errorf("check %q: timeout is not applicable to systemd checks", c.Name)
			}
		case HealthTCP:
			if c.Host == "" {
				return nil, fmt.Errorf("check %q: host is required for tcp checks", c.Name)
			}
			if c.Port == 0 {
				return nil, fmt.Errorf("check %q: port is required for tcp checks", c.Name)
			}
			if c.Port < 1 || c.Port > 65535 {
				return nil, fmt.Errorf("check %q: port %d out of range (1-65535)", c.Name, c.Port)
			}
			if c.Timeout != "" {
				if _, err := time.ParseDuration(c.Timeout); err != nil {
					return nil, fmt.Errorf("check %q: invalid timeout %q (e.g. \"500ms\", \"5s\")", c.Name, c.Timeout)
				}
			}
		case HealthCommand:
			if c.Command == "" {
				return nil, fmt.Errorf("check %q: command is required for command checks", c.Name)
			}
			if c.Timeout != "" {
				if _, err := time.ParseDuration(c.Timeout); err != nil {
					return nil, fmt.Errorf("check %q: invalid timeout %q (e.g. \"500ms\", \"5s\")", c.Name, c.Timeout)
				}
			}
		case HealthHTTP:
			if c.URL == "" {
				return nil, fmt.Errorf("check %q: url is required for http checks", c.Name)
			}
			if u, err := url.Parse(c.URL); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
				return nil, fmt.Errorf("check %q: url must begin with http:// or https://", c.Name)
			}
			if c.Timeout != "" {
				if _, err := time.ParseDuration(c.Timeout); err != nil {
					return nil, fmt.Errorf("check %q: invalid timeout %q (e.g. \"500ms\", \"5s\")", c.Name, c.Timeout)
				}
			}
		case "":
			return nil, fmt.Errorf("check %q: type is required", c.Name)
		default:
			return nil, fmt.Errorf("check %q: unknown type %q (valid: systemd, tcp, command, http)", c.Name, c.Type)
		}
	}

	return &cfg, nil
}
