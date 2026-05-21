package scanner

import (
	"context"
	"errors"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/F0903/traefik-pve-provider/proxmox"
	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
)

func (s *Scanner) ResolveIPs(ctx context.Context, workloads []*inventory.Workload) {
	if s.options.SkipIPResolution {
		return
	}

	workloads = ipResolutionTargets(workloads)
	if len(workloads) == 0 {
		return
	}

	if len(workloads) == 1 || s.maxConcurrency <= 1 {
		for _, workload := range workloads {
			s.resolveWorkloadIPs(ctx, workload)
		}
		return
	}

	limit := minInt(s.maxConcurrency, len(workloads))

	var wg sync.WaitGroup
	sem := make(chan struct{}, limit)
	for _, workload := range workloads {
		sem <- struct{}{}
		// Can't use wg.Go due to Yaegi
		wg.Add(1)
		go func(workload *inventory.Workload) {
			defer wg.Done()
			defer func() { <-sem }()
			s.resolveWorkloadIPs(ctx, workload)
		}(workload)
	}
	wg.Wait()
}

func (s *Scanner) resolveWorkloadIPs(ctx context.Context, workload *inventory.Workload) {
	interfacePatterns, explicitInterfaceFilter := s.interfacePatternsFor(workload)
	hasInterfaceFilter := len(interfacePatterns) > 0
	clearFallbackIPs := explicitInterfaceFilter && hasInterfaceFilter
	switch workload.Kind {
	case inventory.KindVM:
		interfaces, err := s.api.VMNetworkInterfaces(ctx, workload.Node, workload.ID)
		if err != nil {
			if clearFallbackIPs {
				workload.IPs = nil
			}
			workload.Problems = append(workload.Problems, interfaceProblem(workload.Node, workload.Kind, workload.ID, err))
			return
		}
		if ips := ipsFromInterfaces(interfaces.Result, s.ipMode, interfacePatterns); len(ips) > 0 {
			workload.IPs = ips
		} else if clearFallbackIPs {
			workload.IPs = nil
		}
	case inventory.KindContainer:
		interfaces, err := s.api.ContainerInterfaces(ctx, workload.Node, workload.ID)
		if err != nil {
			if clearFallbackIPs {
				workload.IPs = nil
			}
			workload.Problems = append(workload.Problems, interfaceProblem(workload.Node, workload.Kind, workload.ID, err))
			return
		}
		if ips := ipsFromInterfaces(interfaces, s.ipMode, interfacePatterns); len(ips) > 0 {
			workload.IPs = ips
		} else if clearFallbackIPs {
			workload.IPs = nil
		}
	}
}

func (s *Scanner) interfacePatternsFor(workload *inventory.Workload) ([]string, bool) {
	patterns := normalizedInterfacePatterns(workload.InterfacePatterns)
	if len(patterns) > 0 {
		return patterns, true
	}
	return s.defaultInterfaces, false
}

type workloadKey struct {
	node string
	kind inventory.Kind
	id   int
}

func ipResolutionTargets(workloads []*inventory.Workload) []*inventory.Workload {
	seen := make(map[workloadKey]bool, len(workloads))
	targets := make([]*inventory.Workload, 0, len(workloads))
	for _, workload := range workloads {
		if workload == nil || workload.Status != "running" {
			continue
		}

		key := workloadKey{
			node: workload.Node,
			kind: workload.Kind,
			id:   workload.ID,
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		targets = append(targets, workload)
	}
	return targets
}

func ipsFromInterfaces(interfaces []proxmox.NetworkInterface, mode IPMode, interfacePatterns []string) []inventory.IP {
	interfacePatterns = normalizedInterfacePatterns(interfacePatterns)
	seen := make(map[string]bool)
	ips := make([]inventory.IP, 0)

	for _, iface := range interfaces {
		if !allowedGuestInterface(iface.Name, interfacePatterns) {
			continue
		}
		for _, ipAddress := range iface.IPAddresses {
			ip := ipFromAddress(ipAddress.Address, ipAddress.Prefix.String(), iface.Name, mode)
			if ip == nil || seen[ip.Address] {
				continue
			}
			seen[ip.Address] = true
			ips = append(ips, *ip)
		}

		for _, cidr := range []string{iface.Inet, iface.Inet6} {
			ip := ipFromAddress(cidrAddress(cidr), cidrPrefix(cidr), iface.Name, mode)
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

func ipsFromGuestConfig(configs map[string]string, mode IPMode) []inventory.IP {
	seen := make(map[string]bool)
	ips := make([]inventory.IP, 0)

	keys := make([]string, 0, len(configs))
	for key := range configs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		for _, field := range strings.Split(configs[key], ",") {
			name, value, ok := strings.Cut(field, "=")
			if !ok {
				continue
			}
			name = strings.ToLower(strings.TrimSpace(name))
			if name != "ip" && name != "ip6" {
				continue
			}
			ip := ipFromAddress(cidrAddress(value), cidrPrefix(value), key, mode)
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

func ipFromAddress(address, prefix, iface string, mode IPMode) *inventory.IP {
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
	if !mode.allows(version) {
		return nil
	}

	prefixBits := 0
	if prefix != "" {
		if parsedPrefix, err := strconv.Atoi(prefix); err == nil {
			prefixBits = parsedPrefix
		}
	}

	return &inventory.IP{
		Address:   stripZone(address),
		Version:   version,
		Prefix:    prefixBits,
		Interface: iface,
	}
}

func isRoutableGuestIP(ip net.IP) bool {
	return !ip.IsLoopback() && !ip.IsUnspecified() && !ip.IsLinkLocalUnicast() && !ip.IsMulticast()
}

func allowedGuestInterface(name string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		if matchGlob(pattern, name) {
			return true
		}
	}
	return false
}

func normalizedInterfacePatterns(patterns []string) []string {
	normalized := make([]string, 0, len(patterns))
	seen := make(map[string]bool, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || seen[pattern] {
			continue
		}
		seen[pattern] = true
		normalized = append(normalized, pattern)
	}
	return normalized
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

func interfaceProblem(node string, kind inventory.Kind, id int, err error) inventory.Problem {
	message := err.Error()
	var apiErr *proxmox.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == 403 {
		message += "; check that the Proxmox API token has interface discovery privileges such as VM.GuestAgent.Audit on PVE 9+"
	}
	return inventory.Problem{
		Node:    node,
		Kind:    kind,
		ID:      id,
		Stage:   "interfaces",
		Message: message,
	}
}
