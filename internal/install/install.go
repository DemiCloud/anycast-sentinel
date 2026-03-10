package install

import (
	"fmt"
	"os"
	"path/filepath"
)

const serviceTemplate = `[Unit]
Description=Anycast sentinel instance %%i
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=%s run --config /etc/anycast/%%i.toml
User=root
Group=root

[Install]
WantedBy=multi-user.target
`

const timerTemplate = `[Unit]
Description=Periodic Anycast sentinel instance %%i

[Timer]
OnBootSec=30s
OnUnitActiveSec=10s
AccuracySec=1s
Persistent=true
Unit=anycast-sentinel@%%i.service

[Install]
WantedBy=timers.target
`

func InstallSystemd(unitDir, binPath string) error {
	svcPath := filepath.Join(unitDir, "anycast-sentinel@.service")
	tmrPath := filepath.Join(unitDir, "anycast-sentinel@.timer")

	if err := os.WriteFile(svcPath, []byte(fmt.Sprintf(serviceTemplate, binPath)), 0o644); err != nil {
		return fmt.Errorf("write service unit: %w", err)
	}
	if err := os.WriteFile(tmrPath, []byte(timerTemplate), 0o644); err != nil {
		return fmt.Errorf("write timer unit: %w", err)
	}
	return nil
}
