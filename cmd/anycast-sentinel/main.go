package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/demicloud/anycast-sentinel/internal/config"
	"github.com/demicloud/anycast-sentinel/internal/install"
	"github.com/demicloud/anycast-sentinel/internal/run"
	"github.com/demicloud/anycast-sentinel/internal/version"
)

func main() {
	var (
		flagConfig  string
		flagInstall string
		flagHelp    bool
		flagVersion bool
	)

	pflag.StringVarP(&flagConfig, "config", "c", "", "Path to configuration file")
	pflag.StringVarP(&flagInstall, "install", "i", "", "Install systemd templates and enable instance")
	pflag.BoolVarP(&flagHelp, "help", "h", false, "Show help")
	pflag.BoolVarP(&flagVersion, "version", "V", false, "Show version information")

	pflag.Parse()

	switch {
	case flagHelp:
		usage()
		return

	case flagVersion:
		version.Print()
		return

	case flagInstall != "":
		exe, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error resolving executable path: %v\n", err)
			os.Exit(1)
		}
		exe, err = filepath.EvalSymlinks(exe)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error resolving executable symlink: %v\n", err)
			os.Exit(1)
		}

		if err := install.InstallInstance(flagInstall, exe); err != nil {
			fmt.Fprintf(os.Stderr, "install error: %v\n", err)
			os.Exit(1)
		}
		return

	case flagConfig != "":
		cfg, err := config.Load(flagConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "config error: %v\n", err)
			os.Exit(1)
		}

		if err := run.Execute(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "runtime error: %v\n", err)
			os.Exit(1)
		}
		return

	default:
		fmt.Fprintf(os.Stderr, "no mode specified. Use --config or --install.\n")
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`anycast-sentinel

Usage:
  anycast-sentinel --config /etc/anycast/<instance>.toml
  anycast-sentinel --install <instance>
  anycast-sentinel --version
  anycast-sentinel --help

Options:
  -c, --config   Path to configuration file
  -i, --install  Install systemd templates and enable instance
  -V, --version  Show version information
  -h, --help     Show help`)
}
