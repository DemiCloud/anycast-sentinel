package health

import (
	"context"
	"fmt"
	"net"
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
		if err := e.checkOne(ctx, &hc); err != nil {
			return fmt.Errorf("healthcheck %q failed: %w", hc.Name, err)
		}
	}
	return nil
}

func (e *Engine) checkOne(ctx context.Context, hc *config.HealthCheck) error {
	switch hc.Type {
	case config.HealthSystemd:
		return e.checkSystemd(ctx, hc)
	case config.HealthTCP:
		return e.checkTCP(ctx, hc)
	default:
		return fmt.Errorf("unknown healthcheck type: %s", hc.Type)
	}
}

func (e *Engine) checkSystemd(ctx context.Context, hc *config.HealthCheck) error {
	if hc.Unit == "" {
		return fmt.Errorf("systemd healthcheck %q missing unit", hc.Name)
	}
	active, err := e.systemd.IsActive(ctx, hc.Unit)
	if err != nil {
		return err
	}
	if !active {
		return fmt.Errorf("unit %s not active", hc.Unit)
	}
	return nil
}

func (e *Engine) checkTCP(ctx context.Context, hc *config.HealthCheck) error {
	if hc.Host == "" || hc.Port == 0 {
		return fmt.Errorf("tcp healthcheck %q missing host/port", hc.Name)
	}
	timeout := time.Duration(hc.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", hc.Host, hc.Port))
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
