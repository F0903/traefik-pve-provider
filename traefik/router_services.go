package traefik

import (
	"fmt"
	"slices"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
)

func (b *configBuilder) defaultHTTPServiceForRouter(workload inventory.Workload, labels *labelcfg.Set, routerName string, serviceNames []string) string {
	return b.defaultServiceForRouter(workload, labels.HTTP.Routers[routerName], routerName, serviceNames, "HTTP")
}

func (b *configBuilder) defaultTCPServiceForRouter(workload inventory.Workload, labels *labelcfg.Set, routerName string, serviceNames []string) string {
	return b.defaultServiceForRouter(workload, labels.TCP.Routers[routerName], routerName, serviceNames, "TCP")
}

func (b *configBuilder) defaultUDPServiceForRouter(workload inventory.Workload, labels *labelcfg.Set, routerName string, serviceNames []string) string {
	return b.defaultServiceForRouter(workload, labels.UDP.Routers[routerName], routerName, serviceNames, "UDP")
}

func (b *configBuilder) defaultServiceForRouter(workload inventory.Workload, source *labelcfg.Resource, routerName string, serviceNames []string, protocol string) string {
	if source != nil {
		if _, ok := source.StringValue("service"); ok {
			return serviceNames[0]
		}
	}
	if slices.Contains(serviceNames, routerName) {
		return routerName
	}
	if len(serviceNames) > 1 {
		b.addDiagnostic(workload, fmt.Sprintf("%s router %q has no explicit service and no matching service name; using %q", protocol, routerName, serviceNames[0]))
	}
	return serviceNames[0]
}
