package install

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	servicePath = "/etc/systemd/system/anycast-sentinel@.service"
	timerPath   = "/etc/systemd/system/anycast-sentinel@.timer"
	configDir   = "/etc/anycast"
)

var serviceTemplate = `[Unit]
Description=Anycast Sentinel (%i)
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart={{.ExecPath}} run --config /etc/anycast/%i.toml

# Hardening: only CAP_NET_ADMIN is needed for netlink address management.
# To relax a directive, create a drop-in at:
#   /etc/systemd/system/anycast-sentinel@.service.d/override.conf
#
# Example: if a command check uses an interpreter (Python, Ruby, etc.) that
# needs write+execute memory, disable MemoryDenyWriteExecute:
#   [Service]
#   MemoryDenyWriteExecute=no
CapabilityBoundingSet=CAP_NET_ADMIN
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
RestrictAddressFamilies=AF_UNIX AF_NETLINK AF_INET AF_INET6
LockPersonality=yes
MemoryDenyWriteExecute=yes
RestrictNamespaces=yes
RestrictRealtime=yes
`

var timerTemplate = `[Unit]
Description=Anycast Sentinel Timer (%i)

[Timer]
OnBootSec={{.BootDelay}}
OnUnitActiveSec={{.Interval}}
AccuracySec=1s

[Install]
WantedBy=timers.target
`

type tmplData struct {
	ExecPath  string
	Interval  string
	BootDelay string
}

func InstallInstance(instance, execPath, interval, bootDelay string) error {
	fmt.Printf("install: creating config directory %s\n", configDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data := tmplData{ExecPath: execPath, Interval: interval, BootDelay: bootDelay}

	if err := writeTemplateIfChanged(servicePath, serviceTemplate, data); err != nil {
		return err
	}
	if err := writeTemplateIfChanged(timerPath, timerTemplate, data); err != nil {
		return err
	}

	cfg := filepath.Join(configDir, instance+".toml")
	if _, err := os.Stat(cfg); os.IsNotExist(err) {
		fmt.Printf("install: writing sample config %s\n", cfg)
		if err := os.WriteFile(cfg, []byte(sampleConfig()), 0644); err != nil {
			return fmt.Errorf("writing sample config: %w", err)
		}
	} else {
		fmt.Printf("install: config %s already exists, skipping\n", cfg)
	}

	fmt.Println("install: reloading systemd daemon")
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	timer := fmt.Sprintf("anycast-sentinel@%s.timer", instance)
	fmt.Printf("install: enabling and starting %s\n", timer)
	if out, err := exec.Command("systemctl", "enable", "--now", timer).CombinedOutput(); err != nil {
		return fmt.Errorf("enabling timer %s: %w\n%s", timer, err, strings.TrimSpace(string(out)))
	}

	fmt.Printf("install: done — edit %s to configure\n", cfg)
	return nil
}

func writeTemplateIfChanged(path, tmpl string, data tmplData) error {
	t := template.Must(template.New("unit").Parse(tmpl))

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}
	rendered := buf.Bytes()

	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(bytes.TrimSpace(existing), bytes.TrimSpace(rendered)) {
		fmt.Printf("install: %s unchanged, skipping\n", path)
		return nil
	}

	fmt.Printf("install: writing %s\n", path)
	return os.WriteFile(path, rendered, 0644)
}

// UninstallInstance disables and stops the named timer instance, removes the
// shared template unit files, and reloads the systemd daemon.
// The instance config file in /etc/anycast/ is intentionally left in place.
func UninstallInstance(instance string) error {
	timer := fmt.Sprintf("anycast-sentinel@%s.timer", instance)

	fmt.Printf("uninstall: disabling %s\n", timer)
	out, err := exec.Command("systemctl", "disable", "--now", timer).CombinedOutput()
	if err != nil {
		// Non-fatal: the unit may already be stopped or never enabled.
		fmt.Printf("uninstall: note: %s\n", strings.TrimSpace(string(out)))
	}

	for _, path := range []string{servicePath, timerPath} {
		if err := removeFileIfExists(path); err != nil {
			return err
		}
	}

	fmt.Println("uninstall: reloading systemd daemon")
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	fmt.Printf("uninstall: done (config %s/%s.toml was not removed)\n", configDir, instance)
	return nil
}

func removeFileIfExists(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %s: %w", path, err)
	}
	fmt.Printf("uninstall: removed %s\n", path)
	return nil
}

func sampleConfig() string {
	return `[general]
dev = "eth0"
# ip4 = "192.0.2.1"   # IPv4 /32 to announce
# ip6 = "2001:db8::1" # IPv6 /128 to announce (optional)

# At least one check is required. All checks must pass (AND semantics).
# Supported types: systemd, tcp, command

# Check that a systemd unit is active
[[checks]]
name = "my-service"
type = "systemd"
unit = "my-service.service"

# Check that a TCP port is reachable
# [[checks]]
# name = "my-service-port"
# type = "tcp"
# host = "127.0.0.1"
# port = 8080
# timeout = "500ms"   # optional, default 500ms

# Check that a command exits 0
# [[checks]]
# name = "custom-check"
# type = "command"
# command = "/usr/local/bin/my-healthcheck.sh"
# timeout = "5s"      # optional, kills the command if exceeded
`
}
