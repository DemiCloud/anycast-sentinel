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
ExecStart={{.ExecPath}} --config /etc/anycast/%i.toml
`

var timerTemplate = `[Unit]
Description=Anycast Sentinel Timer (%i)

[Timer]
OnBootSec=30s
OnUnitActiveSec=5s
AccuracySec=1s

[Install]
WantedBy=timers.target
`

type tmplData struct {
	ExecPath string
}

func InstallInstance(instance, execPath string) error {
	fmt.Printf("install: creating config directory %s\n", configDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	if err := writeTemplateIfChanged(servicePath, serviceTemplate, execPath); err != nil {
		return err
	}
	if err := writeTemplateIfChanged(timerPath, timerTemplate, execPath); err != nil {
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

func writeTemplateIfChanged(path, tmpl, execPath string) error {
	t := template.Must(template.New("unit").Parse(tmpl))

	var buf bytes.Buffer
	if err := t.Execute(&buf, tmplData{ExecPath: execPath}); err != nil {
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

func sampleConfig() string {
	return `[general]
dev = "eth0"
# ip4 = "192.0.2.1"   # IPv4 /32 to announce
# ip6 = "2001:db8::1" # IPv6 /128 to announce (optional)
# interval = "5s"     # informational only; execution frequency is controlled by the systemd timer

[logging]
level = "info"

[[checks]]
name = "example-service"
type = "systemd"
unit = "example.service"
`
}
