package traefik

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	"github.com/traefik/genconf/dynamic"
)

func (b *configBuilder) addHTTPWorkload(workload inventory.Workload) {
	routerNames, hasExplicitRouters := explicitNamesFromLabels(workload.TraefikLabels, "traefik.http.routers.", supportedHTTPRouterOption)
	serviceNames, hasExplicitServices := explicitNamesFromLabels(workload.TraefikLabels, "traefik.http.services.", supportedHTTPServiceLabelOption)
	transportNames, _ := b.explicitObjectNames(workload, "traefik.http.serverstransports.", supportedHTTPServersTransportOption, b.httpTransports, "HTTP servers transport")
	defaultName := defaultObjectName(workload)

	if len(routerNames) == 0 && !hasExplicitRouters {
		routerNames = []string{b.claimDefaultObjectName(defaultName, workload, b.httpRouters, "HTTP router")}
	}
	if len(serviceNames) == 0 && !hasExplicitServices {
		serviceNames = []string{b.claimDefaultObjectName(defaultName, workload, b.httpServices, "HTTP service")}
	}
	if len(routerNames) == 0 {
		b.addDiagnostic(workload, "no usable HTTP router names remain after collision checks")
		return
	}
	if len(serviceNames) == 0 {
		b.addDiagnostic(workload, "no usable HTTP service names remain after collision checks")
		return
	}

	for _, transportName := range transportNames {
		b.config.HTTP.ServersTransports[transportName] = buildHTTPServersTransport(workload.TraefikLabels, transportName)
	}

	for _, serviceName := range serviceNames {
		b.addOrMergeHTTPService(workload, serviceName)
	}

	for _, routerName := range routerNames {
		router := buildHTTPRouter(workload, routerName, b.defaultHTTPServiceForRouter(workload, routerName, serviceNames), b.options)
		b.addOrSetHTTPRouter(workload, routerName, router)
	}
}

func (b *configBuilder) addTCPWorkload(workload inventory.Workload) {
	routerNames, hasExplicitRouters := b.explicitObjectNames(workload, "traefik.tcp.routers.", supportedTCPRouterOption, b.tcpRouters, "TCP router")
	serviceNames, hasExplicitServices := b.explicitObjectNames(workload, "traefik.tcp.services.", supportedTCPServiceLabelOption, b.tcpServices, "TCP service")
	defaultName := defaultObjectName(workload)

	if len(routerNames) == 0 && !hasExplicitRouters {
		routerNames = []string{b.claimDefaultObjectName(defaultName, workload, b.tcpRouters, "TCP router")}
	}
	if len(serviceNames) == 0 && !hasExplicitServices {
		serviceNames = []string{b.claimDefaultObjectName(defaultName, workload, b.tcpServices, "TCP service")}
	}
	if len(routerNames) == 0 {
		b.addDiagnostic(workload, "no usable TCP router names remain after collision checks")
		return
	}
	if len(serviceNames) == 0 {
		b.addDiagnostic(workload, "no usable TCP service names remain after collision checks")
		return
	}

	for _, serviceName := range serviceNames {
		b.config.TCP.Services[serviceName] = buildTCPService(workload, serviceName)
	}
	for _, routerName := range routerNames {
		b.config.TCP.Routers[routerName] = buildTCPRouter(workload, routerName, b.defaultTCPServiceForRouter(workload, routerName, serviceNames))
	}
}

func (b *configBuilder) addUDPWorkload(workload inventory.Workload) {
	routerNames, hasExplicitRouters := b.explicitObjectNames(workload, "traefik.udp.routers.", supportedUDPRouterOption, b.udpRouters, "UDP router")
	serviceNames, hasExplicitServices := b.explicitObjectNames(workload, "traefik.udp.services.", supportedUDPServiceLabelOption, b.udpServices, "UDP service")
	defaultName := defaultObjectName(workload)

	if len(routerNames) == 0 && !hasExplicitRouters {
		routerNames = []string{b.claimDefaultObjectName(defaultName, workload, b.udpRouters, "UDP router")}
	}
	if len(serviceNames) == 0 && !hasExplicitServices {
		serviceNames = []string{b.claimDefaultObjectName(defaultName, workload, b.udpServices, "UDP service")}
	}
	if len(routerNames) == 0 {
		b.addDiagnostic(workload, "no usable UDP router names remain after collision checks")
		return
	}
	if len(serviceNames) == 0 {
		b.addDiagnostic(workload, "no usable UDP service names remain after collision checks")
		return
	}

	for _, serviceName := range serviceNames {
		b.config.UDP.Services[serviceName] = buildUDPService(workload, serviceName)
	}
	for _, routerName := range routerNames {
		b.config.UDP.Routers[routerName] = buildUDPRouter(workload, routerName, b.defaultUDPServiceForRouter(workload, routerName, serviceNames))
	}
}

func (b *configBuilder) explicitObjectNames(workload inventory.Workload, prefix string, supportedOption func(string) bool, claimed map[string]objectOwner, objectKind string) ([]string, bool) {
	names, found := explicitNamesFromLabels(workload.TraefikLabels, prefix, supportedOption)
	if len(names) == 0 {
		return nil, found
	}

	usable := make([]string, 0, len(names))
	for _, name := range names {
		if b.claimObjectName(name, workload, claimed, objectKind) {
			usable = append(usable, name)
		}
	}
	return usable, true
}

func (b *configBuilder) claimDefaultObjectName(name string, workload inventory.Workload, claimed map[string]objectOwner, objectKind string) string {
	if b.claimObjectName(name, workload, claimed, objectKind) {
		return name
	}

	for _, candidate := range defaultNameCollisionCandidates(name, workload) {
		if b.claimObjectName(candidate, workload, claimed, objectKind) {
			b.addDiagnostic(workload, fmt.Sprintf("%s name %q already exists; using %q", objectKind, name, candidate))
			return candidate
		}
	}

	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s-%d-%d", name, workload.ID, index)
		if b.claimObjectName(candidate, workload, claimed, objectKind) {
			b.addDiagnostic(workload, fmt.Sprintf("%s name %q already exists; using %q", objectKind, name, candidate))
			return candidate
		}
	}
}

func (b *configBuilder) claimObjectName(name string, workload inventory.Workload, claimed map[string]objectOwner, objectKind string) bool {
	owner := ownerFor(workload)
	if existing, exists := claimed[name]; exists {
		if existing == owner {
			return true
		}
		b.addDiagnostic(workload, fmt.Sprintf("%s name %q already used by %s; skipping duplicate", objectKind, name, existing.describe()))
		return false
	}
	claimed[name] = owner
	return true
}

func (b *configBuilder) addOrMergeHTTPService(workload inventory.Workload, serviceName string) {
	service := buildHTTPService(workload, serviceName)
	b.applyHTTPServiceShorthands(workload, serviceName, service)
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

func (b *configBuilder) applyHTTPServiceShorthands(workload inventory.Workload, serviceName string, service *dynamic.Service) {
	if service == nil || service.LoadBalancer == nil || service.LoadBalancer.ServersTransport != "" {
		return
	}

	insecureSkipVerify, ok := httpServiceInsecureSkipVerify(workload.TraefikLabels, serviceName)
	if !ok || !insecureSkipVerify {
		return
	}

	service.LoadBalancer.ServersTransport = b.ensureInsecureHTTPServersTransport(workload, serviceName)
}

func (b *configBuilder) ensureInsecureHTTPServersTransport(workload inventory.Workload, serviceName string) string {
	preferred := serviceName + "-insecure"
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
		candidate := fmt.Sprintf("%s-%d", preferred, index)
		if workload.ID != 0 {
			candidate = fmt.Sprintf("%s-%d-%d", preferred, workload.ID, index)
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

func (b *configBuilder) defaultHTTPServiceForRouter(workload inventory.Workload, routerName string, serviceNames []string) string {
	return b.defaultServiceForRouter(workload, routerName, serviceNames, []string{"traefik.http.routers." + routerName + ".service"}, "HTTP")
}

func (b *configBuilder) defaultTCPServiceForRouter(workload inventory.Workload, routerName string, serviceNames []string) string {
	return b.defaultServiceForRouter(workload, routerName, serviceNames, []string{"traefik.tcp.routers." + routerName + ".service", "traefik.tcp.service"}, "TCP")
}

func (b *configBuilder) defaultUDPServiceForRouter(workload inventory.Workload, routerName string, serviceNames []string) string {
	return b.defaultServiceForRouter(workload, routerName, serviceNames, []string{"traefik.udp.routers." + routerName + ".service", "traefik.udp.service"}, "UDP")
}

func (b *configBuilder) defaultServiceForRouter(workload inventory.Workload, routerName string, serviceNames []string, serviceLabels []string, protocol string) string {
	for _, serviceLabel := range serviceLabels {
		if explicitService := strings.TrimSpace(workload.TraefikLabels[serviceLabel]); explicitService != "" {
			return serviceNames[0]
		}
	}
	if containsString(serviceNames, routerName) {
		return routerName
	}
	if len(serviceNames) > 1 {
		b.addDiagnostic(workload, fmt.Sprintf("%s router %q has no explicit service and no matching service name; using %q", protocol, routerName, serviceNames[0]))
	}
	return serviceNames[0]
}

func (b *configBuilder) addDiagnostic(workload inventory.Workload, message string) {
	b.diagnostics = append(b.diagnostics, Diagnostic{
		Node:    workload.Node,
		Kind:    workload.Kind,
		ID:      workload.ID,
		Message: message,
	})
}

func defaultNameCollisionCandidates(name string, workload inventory.Workload) []string {
	candidates := make([]string, 0, 3)
	if workload.ID != 0 {
		candidates = append(candidates, fmt.Sprintf("%s-%d", name, workload.ID))
	}
	if workload.Kind != "" && workload.ID != 0 {
		candidates = append(candidates, fmt.Sprintf("%s-%s-%d", name, workload.Kind, workload.ID))
	}
	if workload.Node != "" && workload.ID != 0 {
		candidates = append(candidates, fmt.Sprintf("%s-%s-%d", name, workload.Node, workload.ID))
	}
	return candidates
}

func ownerFor(workload inventory.Workload) objectOwner {
	return objectOwner{
		Node: workload.Node,
		Kind: workload.Kind,
		ID:   workload.ID,
		Name: workload.Name,
	}
}

func (o objectOwner) describe() string {
	parts := make([]string, 0, 4)
	if o.Node != "" {
		parts = append(parts, "node="+o.Node)
	}
	if o.Kind != "" {
		parts = append(parts, "kind="+string(o.Kind))
	}
	if o.ID != 0 {
		parts = append(parts, "id="+strconv.Itoa(o.ID))
	}
	if o.Name != "" {
		parts = append(parts, "name="+o.Name)
	}
	return strings.Join(parts, " ")
}
