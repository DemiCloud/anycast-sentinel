package run

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"

	"github.com/demicloud/anycast-sentinel/internal/config"
	"github.com/demicloud/anycast-sentinel/internal/health"
	"github.com/demicloud/anycast-sentinel/internal/systemd"
	"github.com/vishvananda/netlink"
)

// Execute runs a single one-shot health evaluation and adds/removes the
// anycast address accordingly. Stateless. AND semantics for all checks.
// If dryRun is true, route changes are logged but not applied.
func Execute(cfg *config.Config, dryRun bool) error {
	globalTimeout := 30 * time.Second
	if cfg.General.Timeout != "" {
		if d, err := time.ParseDuration(cfg.General.Timeout); err == nil {
			globalTimeout = d
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()

	// Only connect to systemd if the config actually has systemd checks.
	var sd *systemd.Client
	for _, c := range cfg.Checks {
		if c.Type == config.HealthSystemd {
			var err error
			sd, err = systemd.New()
			if err != nil {
				return fmt.Errorf("connecting to systemd: %w", err)
			}
			defer sd.Close()
			break
		}
	}

	engine := health.NewEngine(sd)

	if err := engine.AllHealthy(ctx, cfg.Checks); err != nil {
		fmt.Println("health: checks failed")
		return removeRouteIfPresent(cfg, dryRun)
	}

	fmt.Println("health: all checks passed")
	return addRouteIfMissing(cfg, dryRun)
}

// --- Route decision helpers ---

func addRouteIfMissing(cfg *config.Config, dryRun bool) error {
	present, err := checkRoute(cfg)
	if err != nil {
		return err
	}
	target := routeTarget(cfg)
	if present {
		fmt.Printf("route [%s]: present → keeping\n", target)
		return nil
	}
	if dryRun {
		fmt.Printf("route [%s]: absent → adding (dry run)\n", target)
		return nil
	}
	fmt.Printf("route [%s]: absent → adding\n", target)
	return addRoute(cfg)
}

func removeRouteIfPresent(cfg *config.Config, dryRun bool) error {
	present, err := checkRoute(cfg)
	if err != nil {
		return err
	}
	target := routeTarget(cfg)
	if !present {
		fmt.Printf("route [%s]: absent → nothing to do\n", target)
		return nil
	}
	if dryRun {
		fmt.Printf("route [%s]: present → removing (dry run)\n", target)
		return nil
	}
	fmt.Printf("route [%s]: present → removing\n", target)
	return removeRoute(cfg)
}

// routeTarget formats the device and configured IP(s) for log output.
func routeTarget(cfg *config.Config) string {
	ips := cfg.General.IP4
	if cfg.General.IP6 != "" {
		if ips != "" {
			ips += ", " + cfg.General.IP6
		} else {
			ips = cfg.General.IP6
		}
	}
	return cfg.General.Dev + "/" + ips
}

// --- Netlink helpers ---

// checkRoute returns true only if ALL configured addresses are present on the interface.
func checkRoute(cfg *config.Config) (bool, error) {
	link, err := netlink.LinkByName(cfg.General.Dev)
	if err != nil {
		return false, fmt.Errorf("link %s: %w", cfg.General.Dev, err)
	}

	if cfg.General.IP4 != "" {
		addr, err := parseAddr(cfg.General.IP4, 32)
		if err != nil {
			return false, err
		}
		ok, err := addrOnLink(link, addr)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}

	if cfg.General.IP6 != "" {
		addr, err := parseAddr(cfg.General.IP6, 128)
		if err != nil {
			return false, err
		}
		ok, err := addrOnLink(link, addr)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}

	return true, nil
}

func addRoute(cfg *config.Config) error {
	link, err := netlink.LinkByName(cfg.General.Dev)
	if err != nil {
		return fmt.Errorf("link %s: %w", cfg.General.Dev, err)
	}

	if cfg.General.IP4 != "" {
		addr, err := parseAddr(cfg.General.IP4, 32)
		if err != nil {
			return err
		}
		if err := netlink.AddrAdd(link, addr); err != nil && !isExists(err) {
			return fmt.Errorf("adding IPv4 addr: %w", err)
		}
	}

	if cfg.General.IP6 != "" {
		addr, err := parseAddr(cfg.General.IP6, 128)
		if err != nil {
			return err
		}
		if err := netlink.AddrAdd(link, addr); err != nil && !isExists(err) {
			return fmt.Errorf("adding IPv6 addr: %w", err)
		}
	}

	return nil
}

func removeRoute(cfg *config.Config) error {
	link, err := netlink.LinkByName(cfg.General.Dev)
	if err != nil {
		return fmt.Errorf("link %s: %w", cfg.General.Dev, err)
	}

	if cfg.General.IP4 != "" {
		addr, err := parseAddr(cfg.General.IP4, 32)
		if err != nil {
			return err
		}
		if err := netlink.AddrDel(link, addr); err != nil && !isNotFound(err) {
			return fmt.Errorf("removing IPv4 addr: %w", err)
		}
	}

	if cfg.General.IP6 != "" {
		addr, err := parseAddr(cfg.General.IP6, 128)
		if err != nil {
			return err
		}
		if err := netlink.AddrDel(link, addr); err != nil && !isNotFound(err) {
			return fmt.Errorf("removing IPv6 addr: %w", err)
		}
	}

	return nil
}

// --- Address helpers ---

func parseAddr(ipStr string, bits int) (*netlink.Addr, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP: %s", ipStr)
	}

	var ipNet *net.IPNet
	if ip4 := ip.To4(); ip4 != nil {
		ipNet = &net.IPNet{IP: ip4, Mask: net.CIDRMask(bits, 32)}
	} else {
		ipNet = &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, 128)}
	}

	return &netlink.Addr{IPNet: ipNet}, nil
}

func addrOnLink(link netlink.Link, addr *netlink.Addr) (bool, error) {
	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return false, err
	}

	for _, a := range addrs {
		if a.IPNet == nil {
			continue
		}
		if a.IPNet.IP.Equal(addr.IPNet.IP) && bytes.Equal(a.IPNet.Mask, addr.IPNet.Mask) {
			return true, nil
		}
	}

	return false, nil
}

func isExists(err error) bool {
	return errors.Is(err, syscall.EEXIST)
}

func isNotFound(err error) bool {
	return errors.Is(err, syscall.EADDRNOTAVAIL)
}

// Status reports whether each configured anycast address is currently present on the interface.
// Each address is reported individually. Returns an error if any address is absent, which
// also causes a non-zero exit code when called from the status subcommand.
// No health checks are evaluated and no route changes are made.
func Status(cfg *config.Config) error {
	link, err := netlink.LinkByName(cfg.General.Dev)
	if err != nil {
		return fmt.Errorf("link %s: %w", cfg.General.Dev, err)
	}

	allPresent := true

	if cfg.General.IP4 != "" {
		addr, err := parseAddr(cfg.General.IP4, 32)
		if err != nil {
			return err
		}
		ok, err := addrOnLink(link, addr)
		if err != nil {
			return err
		}
		if ok {
			fmt.Printf("route [%s/%s]: present\n", cfg.General.Dev, cfg.General.IP4)
		} else {
			fmt.Printf("route [%s/%s]: absent\n", cfg.General.Dev, cfg.General.IP4)
			allPresent = false
		}
	}

	if cfg.General.IP6 != "" {
		addr, err := parseAddr(cfg.General.IP6, 128)
		if err != nil {
			return err
		}
		ok, err := addrOnLink(link, addr)
		if err != nil {
			return err
		}
		if ok {
			fmt.Printf("route [%s/%s]: present\n", cfg.General.Dev, cfg.General.IP6)
		} else {
			fmt.Printf("route [%s/%s]: absent\n", cfg.General.Dev, cfg.General.IP6)
			allPresent = false
		}
	}

	if !allPresent {
		return fmt.Errorf("one or more addresses absent from %s", cfg.General.Dev)
	}
	return nil
}
