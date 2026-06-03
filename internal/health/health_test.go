package health

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/demicloud/anycast-sentinel/internal/config"
)

// generateSelfSignedCert creates an in-memory self-signed TLS certificate for testing.
func generateSelfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	return tls.X509KeyPair(certPEM, keyPEM)
}

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
	// Bind a listener, record its port, close it, then wait until the OS has
	// torn down the socket so that subsequent dials receive ECONNREFUSED.
	ln := startTCPListener(t)
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	// Poll until the port is definitely not accepting connections (up to 1s).
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 50*time.Millisecond)
		if err != nil {
			break // port is closed; proceed with the test
		}
		c.Close()
		time.Sleep(10 * time.Millisecond)
	}

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
	checks := []config.HealthCheck{{Name: "bad", Type: "grpc"}}
	if err := e.AllHealthy(context.Background(), checks); err == nil {
		t.Fatal("expected failure for unknown check type")
	}
}
// --- HTTP checks ---

// startHTTPServer starts a simple HTTP server on a random port and returns its URL.
// statusCode controls the response status. The server is shut down when the test ends.
func startHTTPServer(t *testing.T, statusCode int) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
	})
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln) //nolint:errcheck
	t.Cleanup(func() { srv.Close() })
	return "http://" + ln.Addr().String()
}

func TestCheckHTTP_Pass(t *testing.T) {
	base := startHTTPServer(t, http.StatusOK)
	e := NewEngine(nil)
	checks := []config.HealthCheck{{
		Name: "api",
		Type: config.HealthHTTP,
		URL:  base + "/health",
	}}
	if err := e.AllHealthy(context.Background(), checks); err != nil {
		t.Fatalf("expected pass for 200 response, got: %v", err)
	}
}

func TestCheckHTTP_Non2xx(t *testing.T) {
	base := startHTTPServer(t, http.StatusServiceUnavailable)
	e := NewEngine(nil)
	checks := []config.HealthCheck{{
		Name: "api",
		Type: config.HealthHTTP,
		URL:  base + "/health",
	}}
	if err := e.AllHealthy(context.Background(), checks); err == nil {
		t.Fatal("expected failure for 503 response")
	}
}

func TestCheckHTTP_Unreachable(t *testing.T) {
	// Bind and immediately close so the port is definitely not listening.
	ln := startTCPListener(t)
	addr := ln.Addr().String()
	ln.Close()

	e := NewEngine(nil)
	checks := []config.HealthCheck{{
		Name:    "api",
		Type:    config.HealthHTTP,
		URL:     "http://" + addr,
		Timeout: "200ms",
	}}
	if err := e.AllHealthy(context.Background(), checks); err == nil {
		t.Fatal("expected failure for unreachable host")
	}
}

func TestCheckHTTPS_InsecureSkipTLS(t *testing.T) {
	// Start a TLS server with a self-signed certificate.
	// InsecureSkipTLS=true must allow the check to pass despite the untrusted cert.
	cert, err := generateSelfSignedCert()
	if err != nil {
		t.Fatalf("failed to generate self-signed cert: %v", err)
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln) //nolint:errcheck
	t.Cleanup(func() { srv.Close() })

	url := "https://" + ln.Addr().String()

	// Without InsecureSkipTLS the check should fail due to cert verification.
	e := NewEngine(nil)
	checks := []config.HealthCheck{{
		Name:    "tls",
		Type:    config.HealthHTTP,
		URL:     url,
		Timeout: "2s",
	}}
	if err := e.AllHealthy(context.Background(), checks); err == nil {
		t.Fatal("expected failure for self-signed cert without InsecureSkipTLS")
	}

	// With InsecureSkipTLS the check should pass.
	checks[0].InsecureSkipTLS = true
	if err := e.AllHealthy(context.Background(), checks); err != nil {
		t.Fatalf("expected pass with InsecureSkipTLS=true, got: %v", err)
	}
}