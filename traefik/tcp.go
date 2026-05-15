package traefik

import (
	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
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

	if rule, ok := source.StringValue(lexer.TokenRule); ok {
		router.Rule = rule
	}
	if service, ok := source.StringValue(lexer.TokenService); ok {
		router.Service = service
	}
	if entrypoints, ok := source.ListValue(lexer.TokenEntryPoints); ok {
		router.EntryPoints = entrypoints
	} else if entrypoint, ok := source.StringValue(lexer.TokenEntryPoint); ok {
		router.EntryPoints = []string{entrypoint}
	}
	if middlewares, ok := source.ListValue(lexer.TokenMiddlewares); ok {
		router.Middlewares = middlewares
	}
	if priority, ok := source.IntValue(lexer.TokenPriority); ok {
		router.Priority = priority
	}
	router.TLS = buildTCPRouterTLS(source)
	return router
}

func buildTCPService(workload inventory.Workload, source *labelcfg.Resource) *dynamic.TCPService {
	loadBalancer := &dynamic.TCPServersLoadBalancer{
		Servers: buildTCPServers(workload, source),
	}

	if terminationDelay, ok := source.IntValue(lexer.TokenLoadBalancer, lexer.TokenTerminationDelay); ok {
		loadBalancer.TerminationDelay = &terminationDelay
	}
	if proxyProtocolVersion, ok := source.IntValue(lexer.TokenLoadBalancer, lexer.TokenProxyProtocol, lexer.TokenVersion); ok {
		loadBalancer.ProxyProtocol = &dynamic.ProxyProtocol{Version: proxyProtocolVersion}
	}

	return &dynamic.TCPService{LoadBalancer: loadBalancer}
}

func buildTCPServers(workload inventory.Workload, source *labelcfg.Resource) []dynamic.TCPServer {
	addresses := backendAddresses(workload, source)
	servers := make([]dynamic.TCPServer, 0, len(addresses))
	for _, address := range addresses {
		servers = append(servers, dynamic.TCPServer{Address: address})
	}
	return servers
}
