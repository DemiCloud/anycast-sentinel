package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"

	"github.com/demicloud/anycast-sentinel/internal/config"
	"github.com/demicloud/anycast-sentinel/internal/install"
	"github.com/demicloud/anycast-sentinel/internal/run"
	"github.com/demicloud/anycast-sentinel/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "install":
		installCmd(os.Args[2:])
	case "version", "--version", "-V":
		version.Print()
	case "help", "--help", "-h":
		if len(os.Args) > 2 {
			usageFor(os.Args[2])
		} else {
			usage()
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func runCmd(args []string) {
	fs := pflag.NewFlagSet("run", pflag.ContinueOnError)
	fs.Usage = func() { usageFor("run") }
	flagConfig := fs.StringP("config", "c", "", "Path to configuration file")
	flagDryRun := fs.Bool("dry-run", false, "Evaluate checks but do not modify routes")

	if err := fs.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if *flagConfig == "" {
		fmt.Fprintf(os.Stderr, "run: --config is required\n\n")
		usageFor("run")
		os.Exit(1)
	}

	cfg, err := config.Load(*flagConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	if err := run.Execute(cfg, *flagDryRun); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func installCmd(args []string) {
	fs := pflag.NewFlagSet("install", pflag.ContinueOnError)
	fs.Usage = func() { usageFor("install") }
	flagInterval  := fs.String("interval", "5s", "Timer firing interval (e.g. 5s, 1m)")
	flagBootDelay := fs.String("boot-delay", "30s", "Delay before first run after boot (e.g. 30s, 1m)")

	if err := fs.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "install: instance name required\n\n")
		usageFor("install")
		os.Exit(1)
	}
	instance := fs.Arg(0)

	if _, err := time.ParseDuration(*flagInterval); err != nil {
		fmt.Fprintf(os.Stderr, "install: invalid --interval %q: %v\n", *flagInterval, err)
		os.Exit(1)
	}
	if _, err := time.ParseDuration(*flagBootDelay); err != nil {
		fmt.Fprintf(os.Stderr, "install: invalid --boot-delay %q: %v\n", *flagBootDelay, err)
		os.Exit(1)
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "install: error resolving executable path: %v\n", err)
		os.Exit(1)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "install: error resolving executable symlink: %v\n", err)
		os.Exit(1)
	}

	if err := install.InstallInstance(instance, exe, *flagInterval, *flagBootDelay); err != nil {
		fmt.Fprintf(os.Stderr, "install error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`anycast-sentinel — health-gated anycast address announcer

Usage:
  anycast-sentinel <subcommand> [flags]

Subcommands:
  run      Evaluate health checks and manage the anycast address
  install  Install systemd templates and enable a timer instance
  version  Show version information
  help     Show help for a subcommand

Run 'anycast-sentinel help <subcommand>' for subcommand usage.
`)
}

func usageFor(sub string) {
	switch sub {
	case "run":
		fmt.Print(`anycast-sentinel run

Evaluates all configured health checks and adds or removes the anycast
address accordingly. Exits immediately after the routing decision.

Usage:
  anycast-sentinel run --config <path> [flags]

Flags:
  -c, --config    Path to configuration file (required)
      --dry-run   Evaluate checks but do not modify routes
  -h, --help      Show this help
`)
	case "install":
		fmt.Print(`anycast-sentinel install

Installs systemd template units and enables a timer instance.
Creates /etc/anycast/<instance>.toml if it does not exist.

Usage:
  anycast-sentinel install <instance> [flags]

Arguments:
  <instance>     Instance name — enables anycast-sentinel@<instance>.timer

Flags:
      --interval    Timer firing interval (default: 5s)
      --boot-delay  Delay before first run after boot (default: 30s)
  -h, --help        Show this help
`)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", sub)
		usage()
		os.Exit(1)
	}
}
