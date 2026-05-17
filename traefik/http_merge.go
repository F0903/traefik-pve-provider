package traefik

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
	"github.com/traefik/genconf/dynamic"
)

func (b *configBuilder) addOrMergeHTTPService(workload inventory.Workload, labels *labelcfg.Set, serviceName string) {
	source := labels.HTTP.Services[serviceName]
	service := buildHTTPService(workload, source)
	b.applyHTTPServiceShorthands(workload, source, serviceName, service)
	existing := b.config.HTTP.Services[serviceName]
	if existing == nil {
		b.config.HTTP.Services[serviceName] = service
		b.claimObjectName(serviceName, workload, b.httpServices, "HTTP service")
		return
	}

	if !compatibleHTTPServices(existing, service) {
		b.addDiagnostic(workload, fmt.Sprintf("HTTP service name %q already exists with different non-server settings; skipping duplicate service servers", serviceName))
		return
	}

	existing.LoadBalancer.Servers = mergeHTTPServers(existing.LoadBalancer.Servers, service.LoadBalancer.Servers)
}

func (b *configBuilder) applyHTTPServiceShorthands(workload inventory.Workload, source *labelcfg.Resource, serviceName string, service *dynamic.Service) {
	if service == nil || service.LoadBalancer == nil || service.LoadBalancer.ServersTransport != "" {
		return
	}

	insecureSkipVerify, ok := httpServiceInsecureSkipVerify(source)
	if !ok || !insecureSkipVerify {
		return
	}

	service.LoadBalancer.ServersTransport = b.ensureInsecureHTTPServersTransport(workload, serviceName)
}

func (b *configBuilder) ensureInsecureHTTPServersTransport(workload inventory.Workload, serviceName string) string {
	preferred := generatedNameWithSuffix(serviceName, "insecure")
	if reusableInsecureHTTPServersTransport(b.config.HTTP.ServersTransports[preferred]) {
		return preferred
	}
	if b.config.HTTP.ServersTransports[preferred] == nil && b.claimInsecureHTTPServersTransportName(workload, preferred) {
		b.config.HTTP.ServersTransports[preferred] = buildInsecureHTTPServersTransport()
		return preferred
	}

	for _, candidate := range defaultNameCollisionCandidates(preferred, workload) {
		if reusableInsecureHTTPServersTransport(b.config.HTTP.ServersTransports[candidate]) {
			return candidate
		}
		if b.config.HTTP.ServersTransports[candidate] != nil {
			continue
		}
		if b.claimInsecureHTTPServersTransportName(workload, candidate) {
			b.config.HTTP.ServersTransports[candidate] = buildInsecureHTTPServersTransport()
			b.addDiagnostic(workload, fmt.Sprintf("HTTP servers transport name %q already exists; using %q", preferred, candidate))
			return candidate
		}
	}

	for index := 2; ; index++ {
		candidate := generatedNameWithSuffix(preferred, strconv.Itoa(index))
		if workload.ID != 0 {
			candidate = generatedNameWithSuffix(preferred, strconv.Itoa(workload.ID), strconv.Itoa(index))
		}
		if reusableInsecureHTTPServersTransport(b.config.HTTP.ServersTransports[candidate]) {
			return candidate
		}
		if b.config.HTTP.ServersTransports[candidate] != nil {
			continue
		}
		if b.claimInsecureHTTPServersTransportName(workload, candidate) {
			b.config.HTTP.ServersTransports[candidate] = buildInsecureHTTPServersTransport()
			b.addDiagnostic(workload, fmt.Sprintf("HTTP servers transport name %q already exists; using %q", preferred, candidate))
			return candidate
		}
	}
}

func (b *configBuilder) claimInsecureHTTPServersTransportName(workload inventory.Workload, name string) bool {
	return b.claimObjectName(name, workload, b.httpTransports, "HTTP servers transport")
}

func reusableInsecureHTTPServersTransport(transport *dynamic.ServersTransport) bool {
	return transport != nil && transport.InsecureSkipVerify
}

func (b *configBuilder) addOrSetHTTPRouter(workload inventory.Workload, routerName string, router *dynamic.Router) {
	existing := b.config.HTTP.Routers[routerName]
	if existing == nil {
		b.config.HTTP.Routers[routerName] = router
		b.claimObjectName(routerName, workload, b.httpRouters, "HTTP router")
		return
	}

	if !reflect.DeepEqual(existing, router) {
		b.addDiagnostic(workload, fmt.Sprintf("HTTP router name %q already exists with different settings; skipping duplicate router", routerName))
	}
}

func compatibleHTTPServices(left, right *dynamic.Service) bool {
	if left == nil || right == nil || left.LoadBalancer == nil || right.LoadBalancer == nil {
		return reflect.DeepEqual(left, right)
	}

	leftCopy := *left.LoadBalancer
	rightCopy := *right.LoadBalancer
	leftCopy.Servers = nil
	rightCopy.Servers = nil
	return reflect.DeepEqual(leftCopy, rightCopy)
}

func mergeHTTPServers(existing, incoming []dynamic.Server) []dynamic.Server {
	seen := make(map[string]bool, len(existing)+len(incoming))
	merged := make([]dynamic.Server, 0, len(existing)+len(incoming))
	for _, server := range existing {
		if server.URL == "" || seen[server.URL] {
			continue
		}
		seen[server.URL] = true
		merged = append(merged, server)
	}
	for _, server := range incoming {
		if server.URL == "" || seen[server.URL] {
			continue
		}
		seen[server.URL] = true
		merged = append(merged, server)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].URL < merged[j].URL
	})
	return merged
}
