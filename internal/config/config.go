package config

type HealthCheck struct {
	Name    string `toml:"name"`
	Type    string `toml:"type"` // "systemd", "tcp", "command"
	Unit    string `toml:"unit"`
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
	Timeout string `toml:"timeout"` // e.g. "500ms", "5s" — tcp and command checks only
	Command string `toml:"command"`
}

const (
	HealthSystemd = "systemd"
	HealthTCP     = "tcp"
	HealthCommand = "command"
)

type General struct {
	Dev string `toml:"dev"`
	IP4 string `toml:"ip4"`
	IP6 string `toml:"ip6"`
}

type Config struct {
	General General       `toml:"general"`
	Checks  []HealthCheck `toml:"checks"`
}
