package inventory

import (
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/F0903/traefik-pve-provider/proxmox"
)

func ipsFromInterfaces(interfaces []proxmox.NetworkInterface) []IP {
	seen := make(map[string]bool)
	ips := make([]IP, 0)

	for _, iface := range interfaces {
		for _, ipAddress := range iface.IPAddresses {
			ip := ipFromAddress(ipAddress.Address, ipAddress.Prefix.String(), iface.Name)
			if ip == nil || seen[ip.Address] {
				continue
			}
			seen[ip.Address] = true
			ips = append(ips, *ip)
		}

		for _, cidr := range []string{iface.Inet, iface.Inet6} {
			ip := ipFromAddress(cidrAddress(cidr), cidrPrefix(cidr), iface.Name)
			if ip == nil || seen[ip.Address] {
				continue
			}
			seen[ip.Address] = true
			ips = append(ips, *ip)
		}
	}

	sort.Slice(ips, func(i, j int) bool {
		if ips[i].Version != ips[j].Version {
			return ips[i].Version < ips[j].Version
		}
		return ips[i].Address < ips[j].Address
	})
	return ips
}

func ipFromAddress(address, prefix, iface string) *IP {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil
	}

	parsed := net.ParseIP(stripZone(address))
	if parsed == nil || !isRoutableGuestIP(parsed) {
		return nil
	}

	version := 6
	if parsed.To4() != nil {
		version = 4
	}

	prefixBits := 0
	if prefix != "" {
		if parsedPrefix, err := strconv.Atoi(prefix); err == nil {
			prefixBits = parsedPrefix
		}
	}

	return &IP{
		Address:   stripZone(address),
		Version:   version,
		Prefix:    prefixBits,
		Interface: iface,
	}
}

func isRoutableGuestIP(ip net.IP) bool {
	return !ip.IsLoopback() && !ip.IsUnspecified() && !ip.IsLinkLocalUnicast()
}

func cidrAddress(cidr string) string {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return ""
	}
	ip, _, err := net.ParseCIDR(cidr)
	if err == nil {
		return ip.String()
	}
	return strings.Split(cidr, "/")[0]
}

func cidrPrefix(cidr string) string {
	_, network, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil || network == nil {
		parts := strings.Split(cidr, "/")
		if len(parts) == 2 {
			return parts[1]
		}
		return ""
	}
	ones, _ := network.Mask.Size()
	return strconv.Itoa(ones)
}

func stripZone(address string) string {
	if idx := strings.LastIndex(address, "%"); idx != -1 {
		return address[:idx]
	}
	return address
}
