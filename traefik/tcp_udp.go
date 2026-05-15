package traefik

import (
	"strings"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	"github.com/traefik/genconf/dynamic"
)

func buildTCPRouter(workload inventory.Workload, routerName, defaultService string) *dynamic.TCPRouter {
	prefix := "traefik.tcp.routers." + routerName + "."
	router := &dynamic.TCPRouter{
		Rule:    labelOrDefault(workload.TraefikLabels, prefix+"rule", labelOrDefault(workload.TraefikLabels, "traefik.tcp.rule", "HostSNI(`*`)")),
		Service: firstLabelOrDefault(workload.TraefikLabels, defaultService, prefix+"service", "traefik.tcp.service"),
	}

	if entrypoints := splitCSV(firstLabel(workload.TraefikLabels, prefix+"entrypoints", "traefik.tcp.entrypoints")); len(entrypoints) > 0 {
		router.EntryPoints = entrypoints
	} else if entrypoint := firstLabel(workload.TraefikLabels, prefix+"entrypoint", "traefik.tcp.entrypoint"); entrypoint != "" {
		router.EntryPoints = []string{entrypoint}
	}
	if middlewares := splitCSV(firstLabel(workload.TraefikLabels, prefix+"middlewares", "traefik.tcp.middlewares")); len(middlewares) > 0 {
		router.Middlewares = middlewares
	}
	if priority, ok := parseInt(firstLabel(workload.TraefikLabels, prefix+"priority", "traefik.tcp.priority")); ok {
		router.Priority = priority
	}
	router.TLS = buildTCPRouterTLS(workload.TraefikLabels, prefix)
	return router
}

func buildUDPRouter(workload inventory.Workload, routerName, defaultService string) *dynamic.UDPRouter {
	prefix := "traefik.udp.routers." + routerName + "."
	router := &dynamic.UDPRouter{
		Service: firstLabelOrDefault(workload.TraefikLabels, defaultService, prefix+"service", "traefik.udp.service"),
	}

	if entrypoints := splitCSV(firstLabel(workload.TraefikLabels, prefix+"entrypoints", "traefik.udp.entrypoints")); len(entrypoints) > 0 {
		router.EntryPoints = entrypoints
	} else if entrypoint := firstLabel(workload.TraefikLabels, prefix+"entrypoint", "traefik.udp.entrypoint"); entrypoint != "" {
		router.EntryPoints = []string{entrypoint}
	}
	return router
}

func buildTCPService(workload inventory.Workload, serviceName string) *dynamic.TCPService {
	prefix := "traefik.tcp.services." + serviceName + ".loadbalancer."
	loadBalancer := &dynamic.TCPServersLoadBalancer{
		Servers: buildTCPServers(workload, serviceName),
	}

	if terminationDelay, ok := parseInt(workload.TraefikLabels[prefix+"terminationdelay"]); ok {
		loadBalancer.TerminationDelay = &terminationDelay
	} else if terminationDelay, ok := parseInt(workload.TraefikLabels["traefik.tcp.terminationdelay"]); ok {
		loadBalancer.TerminationDelay = &terminationDelay
	}
	if proxyProtocolVersion, ok := parseInt(workload.TraefikLabels[prefix+"proxyprotocol.version"]); ok {
		loadBalancer.ProxyProtocol = &dynamic.ProxyProtocol{Version: proxyProtocolVersion}
	} else if proxyProtocolVersion, ok := parseInt(workload.TraefikLabels["traefik.tcp.proxyprotocol.version"]); ok {
		loadBalancer.ProxyProtocol = &dynamic.ProxyProtocol{Version: proxyProtocolVersion}
	}

	return &dynamic.TCPService{LoadBalancer: loadBalancer}
}

func buildUDPService(workload inventory.Workload, serviceName string) *dynamic.UDPService {
	return &dynamic.UDPService{
		LoadBalancer: &dynamic.UDPServersLoadBalancer{
			Servers: buildUDPServers(workload, serviceName),
		},
	}
}

func buildTCPServers(workload inventory.Workload, serviceName string) []dynamic.TCPServer {
	addresses := backendAddresses(workload, "traefik.tcp.services."+serviceName+".loadbalancer.server.", "traefik.tcp.")
	servers := make([]dynamic.TCPServer, 0, len(addresses))
	for _, address := range addresses {
		servers = append(servers, dynamic.TCPServer{Address: address})
	}
	return servers
}

func buildUDPServers(workload inventory.Workload, serviceName string) []dynamic.UDPServer {
	addresses := backendAddresses(workload, "traefik.udp.services."+serviceName+".loadbalancer.server.", "traefik.udp.")
	servers := make([]dynamic.UDPServer, 0, len(addresses))
	for _, address := range addresses {
		servers = append(servers, dynamic.UDPServer{Address: address})
	}
	return servers
}

func backendAddresses(workload inventory.Workload, prefix, shorthandPrefix string) []string {
	if address := firstLabel(workload.TraefikLabels, prefix+"address", shorthandPrefix+"address"); address != "" {
		return []string{address}
	}

	port := firstLabel(workload.TraefikLabels, prefix+"port", shorthandPrefix+"port")
	if port == "" {
		port = "80"
	}
	if ip := firstLabel(workload.TraefikLabels, prefix+"ip", shorthandPrefix+"ip"); ip != "" {
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

	return []string{serverAddress(workload.Name+"."+workload.Node, port)}
}
