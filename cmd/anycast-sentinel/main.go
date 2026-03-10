package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/demicloud/anycast-sentinel/internal/config"
	"github.com/demicloud/anycast-sentinel/internal/health"
	"github.com/demicloud/anycast-sentinel/internal/install"
	"github.com/demicloud/anycast-sentinel/internal/netif"
	"github.com/demicloud/anycast-sentinel/internal/systemd"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: anycast-sentinel <run|install-systemd> [flags]")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "install-systemd":
		installCmd(os.Args[2:])
	default:
		fmt.Fprintln(os.Stderr, "unknown subcommand:", os.Args[1])
		os.Exit(2)
	}
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", "", "path to config TOML")
	timeout := fs.Duration("timeout", 3*time.Second, "overall health evaluation timeout")
	_ = fs.Parse(args)

	if *configPath == "" {
		log.Fatal("missing --config")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	sd, err := systemd.New()
	if err != nil {
		log.Fatalf("systemd connection failed: %v", err)
	}
	defer sd.Close()

	engine := health.NewEngine(sd)

	if err := engine.AllHealthy(ctx, cfg.Checks); err != nil {
		log.Printf("health: unhealthy: %v", err)
		changed, err := netif.EnsureIPAbsent(cfg.AnycastDev, cfg.AnycastIP)
		if err != nil {
			log.Fatalf("ip-del failed: %v", err)
		}
		if changed {
			log.Printf("ip-del: %s", cfg.AnycastIP)
		} else {
			log.Printf("ip-del-noop: already-absent")
		}
		return
	}

	log.Printf("health: ok")
	changed, err := netif.EnsureIPPresent(cfg.AnycastDev, cfg.AnycastIP)
	if err != nil {
		log.Fatalf("ip-add failed: %v", err)
	}
	if changed {
		log.Printf("ip-add: %s", cfg.AnycastIP)
	} else {
		log.Printf("ip-add-noop: already-present")
	}
}

func installCmd(args []string) {
	fs := flag.NewFlagSet("install-systemd", flag.ExitOnError)
	unitDir := fs.String("unit-dir", "/etc/systemd/system", "systemd unit directory")
	binPath := fs.String("bin", "/usr/local/sbin/anycast-sentinel", "path to installed binary")
	_ = fs.Parse(args)

	if err := install.InstallSystemd(*unitDir, *binPath); err != nil {
		log.Fatalf("install-systemd failed: %v", err)
	}
	log.Printf("installed systemd units in %s", *unitDir)
}
