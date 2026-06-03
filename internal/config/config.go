package config

type HealthCheck struct {
	Name            string `toml:"name"`
	Type            string `toml:"type"` // "systemd", "tcp", "command", "http"
	Unit            string `toml:"unit"`
	Host            string `toml:"host"`
	Port            int    `toml:"port"`
	Timeout         string `toml:"timeout"` // e.g. "500ms", "5s" — tcp, command, and http checks only
	Command         string `toml:"command"`
	URL             string `toml:"url"`
	InsecureSkipTLS bool   `toml:"insecure_skip_tls"`
}

const (
	HealthSystemd = "systemd"
	HealthTCP     = "tcp"
	HealthCommand = "command"
	HealthHTTP    = "http"
)

type General struct {
	Dev     string `toml:"dev"`
	IP4     string `toml:"ip4"`
	IP6     string `toml:"ip6"`
	Timeout string `toml:"timeout"` // e.g. "30s", "2m" — global deadline for all checks; default 30s
}

type Config struct {
	General General       `toml:"general"`
	Checks  []HealthCheck `toml:"checks"`
}
