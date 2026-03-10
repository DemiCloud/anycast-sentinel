package netif

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

func EnsureIPPresent(dev, cidr string) (bool, error) {
	return ensureIP(dev, cidr, true)
}

func EnsureIPAbsent(dev, cidr string) (bool, error) {
	return ensureIP(dev, cidr, false)
}

func ensureIP(dev, cidr string, present bool) (bool, error) {
	link, err := netlink.LinkByName(dev)
	if err != nil {
		return false, fmt.Errorf("link %s: %w", dev, err)
	}

	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, fmt.Errorf("parse cidr %q: %w", cidr, err)
	}
	addr := &netlink.Addr{IPNet: &net.IPNet{IP: ip, Mask: ipNet.Mask}}

	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return false, fmt.Errorf("list addrs: %w", err)
	}

	has := false
	for _, a := range addrs {
		if a.IPNet == nil {
			continue
		}
		if a.IPNet.IP.Equal(ip) && a.IPNet.Mask.String() == ipNet.Mask.String() {
			has = true
			break
		}
	}

	switch {
	case present && !has:
		if err := netlink.AddrAdd(link, addr); err != nil {
			return false, fmt.Errorf("addr add: %w", err)
		}
		return true, nil
	case !present && has:
		if err := netlink.AddrDel(link, addr); err != nil {
			return false, fmt.Errorf("addr del: %w", err)
		}
		return true, nil
	default:
		return false, nil
	}
}
