package traefik

import (
	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
	"github.com/traefik/genconf/dynamic"
)

func buildUDPRouter(source *labelcfg.Resource, defaultService string) *dynamic.UDPRouter {
	router := &dynamic.UDPRouter{Service: defaultService}
	if source == nil {
		return router
	}

	if service, ok := source.StringValue("service"); ok {
		router.Service = service
	}
	if entrypoints, ok := source.ListValue("entrypoints"); ok {
		router.EntryPoints = entrypoints
	} else if entrypoint, ok := source.StringValue("entrypoint"); ok {
		router.EntryPoints = []string{entrypoint}
	}
	return router
}

func buildUDPService(workload inventory.Workload, source *labelcfg.Resource) *dynamic.UDPService {
	return &dynamic.UDPService{
		LoadBalancer: &dynamic.UDPServersLoadBalancer{
			Servers: buildUDPServers(workload, source),
		},
	}
}

func buildUDPServers(workload inventory.Workload, source *labelcfg.Resource) []dynamic.UDPServer {
	addresses := backendAddresses(workload, source)
	servers := make([]dynamic.UDPServer, 0, len(addresses))
	for _, address := range addresses {
		servers = append(servers, dynamic.UDPServer{Address: address})
	}
	return servers
}
