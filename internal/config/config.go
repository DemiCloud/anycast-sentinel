package config

type HealthCheck struct {
	Name      string `toml:"name"`
	Type      string `toml:"type"` // "systemd", "tcp", "command"
	Unit      string `toml:"unit"`
	Host      string `toml:"host"`
	Port      int    `toml:"port"`
	TimeoutMs int    `toml:"timeout_ms"`
	Command   string `toml:"command"`
}

const (
	HealthSystemd = "systemd"
	HealthTCP     = "tcp"
	HealthCommand = "command"
)

type Health struct {
	Checks []HealthCheck `toml:"checks"`
}

type General struct {
	Dev      string `toml:"dev"`
	IP4      string `toml:"ip4"`
	IP6      string `toml:"ip6"`
	Interval string `toml:"interval"`
}

type Logging struct {
	Level string `toml:"level"`
}

type Config struct {
	General General       `toml:"general"`
	Logging Logging       `toml:"logging"`
	Checks  []HealthCheck `toml:"checks"`
}
