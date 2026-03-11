package health

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/demicloud/anycast-sentinel/internal/config"
	"github.com/demicloud/anycast-sentinel/internal/systemd"
)

type Engine struct {
	systemd *systemd.Client
}

func NewEngine(sd *systemd.Client) *Engine {
	return &Engine{systemd: sd}
}

func (e *Engine) AllHealthy(ctx context.Context, checks []config.HealthCheck) error {
	for _, hc := range checks {
		detail, err := e.checkOne(ctx, &hc)
		if err != nil {
			fmt.Printf("check [%s]: %s → failed (%s)\n", hc.Type, hc.Name, detail)
			return fmt.Errorf("healthcheck %q failed: %w", hc.Name, err)
		}
		fmt.Printf("check [%s]: %s → passed (%s)\n", hc.Type, hc.Name, detail)
	}
	return nil
}

func (e *Engine) checkOne(ctx context.Context, hc *config.HealthCheck) (string, error) {
	switch hc.Type {
	case config.HealthSystemd:
		return e.checkSystemd(ctx, hc)
	case config.HealthTCP:
		return e.checkTCP(ctx, hc)
	case config.HealthCommand:
		return e.checkCommand(ctx, hc)
	default:
		return "unknown type", fmt.Errorf("unknown healthcheck type: %s", hc.Type)
	}
}

func (e *Engine) checkSystemd(ctx context.Context, hc *config.HealthCheck) (string, error) {
	if hc.Unit == "" {
		return "missing unit", fmt.Errorf("systemd healthcheck %q missing unit", hc.Name)
	}
	state, err := e.systemd.ActiveState(ctx, hc.Unit)
	if err != nil {
		return err.Error(), err
	}
	if state != "active" {
		return state, fmt.Errorf("unit %s not active", hc.Unit)
	}
	return state, nil
}

func (e *Engine) checkTCP(ctx context.Context, hc *config.HealthCheck) (string, error) {
	if hc.Host == "" || hc.Port == 0 {
		return "missing host/port", fmt.Errorf("tcp healthcheck %q missing host/port", hc.Name)
	}
	timeout := time.Duration(hc.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", hc.Host, hc.Port))
	if err != nil {
		return err.Error(), err
	}
	_ = conn.Close()
	return "connected", nil
}

func (e *Engine) checkCommand(ctx context.Context, hc *config.HealthCheck) (string, error) {
	if hc.Command == "" {
		return "missing command", fmt.Errorf("command healthcheck %q missing command", hc.Name)
	}
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", hc.Command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if i := strings.IndexByte(detail, '\n'); i >= 0 {
			detail = detail[:i]
		}
		if detail == "" {
			detail = err.Error()
		}
		return detail, fmt.Errorf("command failed: %s", detail)
	}
	return "exit 0", nil
}
