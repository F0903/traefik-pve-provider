package traefik

import (
	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
	"github.com/traefik/genconf/dynamic"
)

func buildTCPRouter(source *labelcfg.Resource, defaultService string) *dynamic.TCPRouter {
	router := &dynamic.TCPRouter{
		Rule:    "HostSNI(`*`)",
		Service: defaultService,
	}
	if source == nil {
		return router
	}

	if rule, ok := source.StringValue("rule"); ok {
		router.Rule = rule
	}
	if service, ok := source.StringValue("service"); ok {
		router.Service = service
	}
	if entrypoints, ok := source.ListValue("entrypoints"); ok {
		router.EntryPoints = entrypoints
	} else if entrypoint, ok := source.StringValue("entrypoint"); ok {
		router.EntryPoints = []string{entrypoint}
	}
	if middlewares, ok := source.ListValue("middlewares"); ok {
		router.Middlewares = middlewares
	}
	if priority, ok := source.IntValue("priority"); ok {
		router.Priority = priority
	}
	router.TLS = buildTCPRouterTLS(source)
	return router
}

func buildTCPService(workload inventory.Workload, source *labelcfg.Resource, options Options) *dynamic.TCPService {
	loadBalancer := &dynamic.TCPServersLoadBalancer{
		Servers: buildTCPServers(workload, source, options),
	}

	if terminationDelay, ok := source.IntValue("loadbalancer.terminationdelay"); ok {
		loadBalancer.TerminationDelay = &terminationDelay
	}
	if proxyProtocolVersion, ok := source.IntValue("loadbalancer.proxyprotocol.version"); ok {
		loadBalancer.ProxyProtocol = &dynamic.ProxyProtocol{Version: proxyProtocolVersion}
	}

	return &dynamic.TCPService{LoadBalancer: loadBalancer}
}

func buildTCPServers(workload inventory.Workload, source *labelcfg.Resource, options Options) []dynamic.TCPServer {
	addresses := backendAddresses(workload, source, options)
	servers := make([]dynamic.TCPServer, 0, len(addresses))
	for _, address := range addresses {
		servers = append(servers, dynamic.TCPServer{Address: address})
	}
	return servers
}
