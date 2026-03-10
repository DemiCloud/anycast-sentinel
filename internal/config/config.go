package config

type LoggingConfig struct {
	Level string `toml:"level"`
}

type HealthCheckType string

const (
	HealthSystemd HealthCheckType = "systemd"
	HealthTCP     HealthCheckType = "tcp"
)

type HealthCheck struct {
	Name string          `toml:"name"`
	Type HealthCheckType `toml:"type"`

	// systemd
	Unit string `toml:"unit,omitempty"`

	// tcp
	Host      string `toml:"host,omitempty"`
	Port      int    `toml:"port,omitempty"`
	TimeoutMs int    `toml:"timeout_ms,omitempty"`
}

type Config struct {
	AnycastIP  string        `toml:"anycast_ip"`
	AnycastDev string        `toml:"anycast_dev"`
	Logging    LoggingConfig `toml:"logging"`
	Checks     []HealthCheck `toml:"healthcheck"`
}
