package config

import (
	"os"
	"strings"
	"testing"
)

// writeTemp writes content to a temporary TOML file and returns its path.
// The file is cleaned up automatically when the test ends.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.toml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		toml    string
		wantErr string // substring that must appear in the error; empty means success
	}{
		// --- valid configurations ---
		{
			name: "valid systemd check",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
`,
		},
		{
			name: "valid ip6 only",
			toml: `
[general]
dev = "eth0"
ip6 = "2001:db8::1"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
`,
		},
		{
			name: "valid dual-stack",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"
ip6 = "2001:db8::1"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
`,
		},
		{
			name: "valid tcp check",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "port"
type = "tcp"
host = "127.0.0.1"
port = 8080
`,
		},
		{
			name: "valid tcp check with timeout",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "port"
type = "tcp"
host = "127.0.0.1"
port = 8080
timeout = "500ms"
`,
		},
		{
			name: "valid command check",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "cmd"
type = "command"
command = "/usr/bin/true"
`,
		},
		{
			name: "valid command check with timeout",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "cmd"
type = "command"
command = "/usr/bin/true"
timeout = "5s"
`,
		},
		{
			name: "valid http check",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "api"
type = "http"
url = "http://127.0.0.1:8080/health"
`,
		},
		{
			name: "valid https check with insecure_skip_tls",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "api"
type = "http"
url = "https://127.0.0.1:8443/health"
insecure_skip_tls = true
timeout = "2s"
`,
		},
		{
			name: "multiple checks of different types",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"

[[checks]]
name = "port"
type = "tcp"
host = "127.0.0.1"
port = 8080

[[checks]]
name = "cmd"
type = "command"
command = "/usr/bin/true"
`,
		},

		// --- general section errors ---
		{
			name: "missing dev",
			toml: `
[general]
ip4 = "203.0.113.1"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
`,
			wantErr: "general.dev is required",
		},
		{
			name: "missing ip4 and ip6",
			toml: `
[general]
dev = "eth0"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
`,
			wantErr: "at least one of general.ip4 or general.ip6 is required",
		},
		{
			name: "invalid ip4",
			toml: `
[general]
dev = "eth0"
ip4 = "not-an-ip"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
`,
			wantErr: "invalid IPv4 address",
		},
		{
			name: "ipv6 address in ip4 field rejected",
			toml: `
[general]
dev = "eth0"
ip4 = "2001:db8::1"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
`,
			wantErr: "invalid IPv4 address",
		},
		{
			name: "invalid ip6",
			toml: `
[general]
dev = "eth0"
ip6 = "not-an-ip"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
`,
			wantErr: "invalid IPv6 address",
		},
		{
			name: "ipv4 address in ip6 field rejected",
			toml: `
[general]
dev = "eth0"
ip6 = "203.0.113.1"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
`,
			wantErr: "invalid IPv6 address",
		},
		{
			name: "valid global timeout",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"
timeout = "2m"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
`,
		},
		{
			name: "global timeout invalid duration",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"
timeout = "not-a-duration"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
`,
			wantErr: "general.timeout: invalid duration",
		},
		{
			name: "global timeout zero rejected",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"
timeout = "0s"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
`,
			wantErr: "general.timeout: invalid duration",
		},
		{
			name: "global timeout negative rejected",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"
timeout = "-5s"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
`,
			wantErr: "general.timeout: invalid duration",
		},

		// --- checks array errors ---
		{
			name: "no checks",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"
`,
			wantErr: "at least one check must be defined",
		},
		{
			name: "check missing name",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
type = "systemd"
unit = "foo.service"
`,
			wantErr: "name is required",
		},
		{
			name: "duplicate check names",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"

[[checks]]
name = "svc"
type = "command"
command = "/usr/bin/true"
`,
			wantErr: "duplicate name",
		},
		{
			name: "check missing type",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "svc"
unit = "foo.service"
`,
			wantErr: "type is required",
		},
		{
			name: "unknown check type",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "svc"
type = "grpc"
`,
			wantErr: "unknown type",
		},

		// --- systemd check errors ---
		{
			name: "systemd check missing unit",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "svc"
type = "systemd"
`,
			wantErr: "unit is required for systemd checks",
		},
		{
			name: "systemd check with timeout is rejected",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
timeout = "5s"
`,
			wantErr: "timeout is not applicable to systemd checks",
		},

		// --- tcp check errors ---
		{
			name: "tcp check missing host",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "port"
type = "tcp"
port = 8080
`,
			wantErr: "host is required for tcp checks",
		},
		{
			name: "tcp check missing port",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "port"
type = "tcp"
host = "127.0.0.1"
`,
			wantErr: "port is required for tcp checks",
		},
		{
			name: "tcp check port too high",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "port"
type = "tcp"
host = "127.0.0.1"
port = 70000
`,
			wantErr: "out of range",
		},
		{
			name: "tcp check port negative",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "port"
type = "tcp"
host = "127.0.0.1"
port = -1
`,
			wantErr: "out of range",
		},
		{
			name: "tcp check invalid timeout",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "port"
type = "tcp"
host = "127.0.0.1"
port = 8080
timeout = "not-a-duration"
`,
			wantErr: "invalid timeout",
		},

		// --- command check errors ---
		{
			name: "command check missing command",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "cmd"
type = "command"
`,
			wantErr: "command is required for command checks",
		},
		{
			name: "command check invalid timeout",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "cmd"
type = "command"
command = "/usr/bin/true"
timeout = "not-a-duration"
`,
			wantErr: "invalid timeout",
		},

		// --- http check errors ---
		{
			name: "http check missing url",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "api"
type = "http"
`,
			wantErr: "url is required for http checks",
		},
		{
			name: "http check invalid url scheme",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "api"
type = "http"
url = "file:///etc/passwd"
`,
			wantErr: "url must begin with http",
		},
		{
			name: "http check url with no scheme",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "api"
type = "http"
url = "example.com/health"
`,
			wantErr: "url must begin with http",
		},
		{
			name: "http check invalid timeout",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"

[[checks]]
name = "api"
type = "http"
url = "http://127.0.0.1:8080/health"
timeout = "not-a-duration"
`,
			wantErr: "invalid timeout",
		},

		// --- TOML structure errors ---
		{
			name: "unknown field rejected",
			toml: `
[general]
dev = "eth0"
ip4 = "203.0.113.1"
bogus_field = "bad"

[[checks]]
name = "svc"
type = "systemd"
unit = "foo.service"
`,
			wantErr: "strict mode",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTemp(t, tc.toml)
			cfg, err := Load(path)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil (cfg=%+v)", tc.wantErr, cfg)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg == nil {
				t.Fatal("Load returned nil config with nil error")
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(t.TempDir() + "/does-not-exist.toml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
