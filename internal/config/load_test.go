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
			wantErr: "invalid IP address",
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
			wantErr: "invalid IP address",
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
type = "http"
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
			wantErr: "bogus_field",
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
