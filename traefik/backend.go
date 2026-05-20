package traefik

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
)

func backendAddresses(workload inventory.Workload, source *labelcfg.Resource, options Options) []string {
	if address, ok := source.StringValue("loadbalancer.server.address"); ok {
		return []string{address}
	}

	port := "80"
	if parsed, ok := source.IntValue("loadbalancer.server.port"); ok {
		port = strconv.Itoa(parsed)
	}
	if ip, ok := source.StringValue("loadbalancer.server.ip"); ok {
		return []string{serverAddress(ip, port)}
	}

	addresses := make([]string, 0, len(workload.IPs))
	for _, ip := range workload.IPs {
		if strings.TrimSpace(ip.Address) == "" {
			continue
		}
		addresses = append(addresses, serverAddress(ip.Address, port))
	}
	if len(addresses) > 0 {
		return addresses
	}

	return []string{serverAddress(fallbackBackendHost(workload, options), port)}
}

func fallbackBackendHost(workload inventory.Workload, options Options) string {
	rawName := strings.TrimSpace(workload.Name)
	name := normalizeGeneratedName(workload.Name)
	if name == "" {
		name = "workload"
	}
	if rawName == "" {
		rawName = name
	}
	if domain := strings.Trim(strings.TrimSpace(options.DefaultDomain), "."); domain != "" {
		return name + "." + domain
	}
	if node := strings.TrimSpace(workload.Node); node != "" {
		return rawName + "." + node
	}
	return rawName
}

func serverURL(scheme, host, port string) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("%s://%s:%s", scheme, host, port)
}

func serverAddress(host, port string) string {
	return net.JoinHostPort(host, port)
}
