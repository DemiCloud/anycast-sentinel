package health

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/demicloud/anycast-sentinel/internal/config"
)

// mockQuerier implements stateQuerier for tests — no real D-Bus connection needed.
type mockQuerier struct {
	state string
	err   error
}

func (m *mockQuerier) ActiveState(_ context.Context, _ string) (string, error) {
	return m.state, m.err
}

// --- systemd checks ---

func TestCheckSystemd_Active(t *testing.T) {
	e := &Engine{systemd: &mockQuerier{state: "active"}}
	checks := []config.HealthCheck{{Name: "svc", Type: config.HealthSystemd, Unit: "foo.service"}}
	if err := e.AllHealthy(context.Background(), checks); err != nil {
		t.Fatalf("expected pass for active unit, got: %v", err)
	}
}

func TestCheckSystemd_Inactive(t *testing.T) {
	e := &Engine{systemd: &mockQuerier{state: "inactive"}}
	checks := []config.HealthCheck{{Name: "svc", Type: config.HealthSystemd, Unit: "foo.service"}}
	if err := e.AllHealthy(context.Background(), checks); err == nil {
		t.Fatal("expected failure for inactive unit")
	}
}

func TestCheckSystemd_DBusError(t *testing.T) {
	e := &Engine{systemd: &mockQuerier{err: fmt.Errorf("dbus: connection refused")}}
	checks := []config.HealthCheck{{Name: "svc", Type: config.HealthSystemd, Unit: "foo.service"}}
	if err := e.AllHealthy(context.Background(), checks); err == nil {
		t.Fatal("expected failure when D-Bus returns an error")
	}
}

func TestCheckSystemd_NilClient(t *testing.T) {
	e := NewEngine(nil) // no systemd connection
	checks := []config.HealthCheck{{Name: "svc", Type: config.HealthSystemd, Unit: "foo.service"}}
	if err := e.AllHealthy(context.Background(), checks); err == nil {
		t.Fatal("expected failure when systemd client is nil")
	}
}

// --- TCP checks ---

// startTCPListener binds a listener on a random port and returns it.
// The caller is responsible for closing it.
func startTCPListener(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

func TestCheckTCP_Pass(t *testing.T) {
	ln := startTCPListener(t)
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	e := NewEngine(nil)
	checks := []config.HealthCheck{{
		Name: "port",
		Type: config.HealthTCP,
		Host: "127.0.0.1",
		Port: port,
	}}
	if err := e.AllHealthy(context.Background(), checks); err != nil {
		t.Fatalf("expected pass for open port, got: %v", err)
	}
}

func TestCheckTCP_Fail(t *testing.T) {
	// Bind then immediately close so the port is definitely not listening.
	ln := startTCPListener(t)
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	e := NewEngine(nil)
	checks := []config.HealthCheck{{
		Name:    "port",
		Type:    config.HealthTCP,
		Host:    "127.0.0.1",
		Port:    port,
		Timeout: "200ms",
	}}
	if err := e.AllHealthy(context.Background(), checks); err == nil {
		t.Fatal("expected failure for closed port")
	}
}

// --- command checks ---

func TestCheckCommand_Pass(t *testing.T) {
	e := NewEngine(nil)
	checks := []config.HealthCheck{{
		Name:    "true",
		Type:    config.HealthCommand,
		Command: "true",
	}}
	if err := e.AllHealthy(context.Background(), checks); err != nil {
		t.Fatalf("expected pass for 'true', got: %v", err)
	}
}

func TestCheckCommand_Fail(t *testing.T) {
	e := NewEngine(nil)
	checks := []config.HealthCheck{{
		Name:    "false",
		Type:    config.HealthCommand,
		Command: "false",
	}}
	if err := e.AllHealthy(context.Background(), checks); err == nil {
		t.Fatal("expected failure for 'false'")
	}
}

func TestCheckCommand_FailOutputInError(t *testing.T) {
	e := NewEngine(nil)
	checks := []config.HealthCheck{{
		Name:    "fail-with-output",
		Type:    config.HealthCommand,
		Command: "echo 'something went wrong'; exit 1",
	}}
	err := e.AllHealthy(context.Background(), checks)
	if err == nil {
		t.Fatal("expected failure")
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Fatalf("expected error to contain command output, got: %v", err)
	}
}

func TestCheckCommand_Timeout(t *testing.T) {
	e := NewEngine(nil)
	checks := []config.HealthCheck{{
		Name:    "sleep",
		Type:    config.HealthCommand,
		Command: "sleep 10",
		Timeout: "50ms",
	}}
	if err := e.AllHealthy(context.Background(), checks); err == nil {
		t.Fatal("expected failure for timed-out command")
	}
}

// --- AllHealthy short-circuit ---

func TestAllHealthy_StopsAtFirstFailure(t *testing.T) {
	// The second check would pass, but AllHealthy should stop after the first failure.
	e := NewEngine(nil)
	checks := []config.HealthCheck{
		{Name: "fail", Type: config.HealthCommand, Command: "false"},
		{Name: "pass", Type: config.HealthCommand, Command: "true"},
	}
	err := e.AllHealthy(context.Background(), checks)
	if err == nil {
		t.Fatal("expected failure")
	}
	// The returned error names the failing check, not the second one.
	if !strings.Contains(err.Error(), `"fail"`) {
		t.Fatalf("expected error to reference 'fail' check, got: %v", err)
	}
}

func TestAllHealthy_AllPass(t *testing.T) {
	e := &Engine{systemd: &mockQuerier{state: "active"}}
	checks := []config.HealthCheck{
		{Name: "svc", Type: config.HealthSystemd, Unit: "foo.service"},
		{Name: "true", Type: config.HealthCommand, Command: "true"},
	}
	if err := e.AllHealthy(context.Background(), checks); err != nil {
		t.Fatalf("expected all checks to pass, got: %v", err)
	}
}

func TestAllHealthy_UnknownType(t *testing.T) {
	e := NewEngine(nil)
	checks := []config.HealthCheck{{Name: "bad", Type: "http"}}
	if err := e.AllHealthy(context.Background(), checks); err == nil {
		t.Fatal("expected failure for unknown check type")
	}
}
