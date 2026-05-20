package traefik

import (
	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
)

func (b *configBuilder) addHTTPWorkload(workload inventory.Workload, names GeneratedNames, labels *labelcfg.Set) {
	routerNames, hasExplicitRouters := labels.HTTP.RouterNames()
	serviceNames, hasExplicitServices := labels.HTTP.ServiceNames()
	transportNames, hasExplicitTransports := labels.HTTP.ServersTransportNames()
	transportNames, _ = b.claimObjectNames(workload, transportNames, hasExplicitTransports, b.httpTransports, "HTTP servers transport")
	defaultName := defaultObjectName(names, labels)

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
		b.config.HTTP.ServersTransports[transportName] = buildHTTPServersTransport(labels.HTTP.ServersTransports[transportName])
	}

	for _, serviceName := range serviceNames {
		b.addOrMergeHTTPService(workload, labels, serviceName)
	}

	for _, routerName := range routerNames {
		router := buildHTTPRouter(labels.HTTP.Routers[routerName], routerName, b.defaultHTTPServiceForRouter(workload, labels, routerName, serviceNames), b.options)
		b.addOrSetHTTPRouter(workload, routerName, router)
	}
}

func (b *configBuilder) addTCPWorkload(workload inventory.Workload, names GeneratedNames, labels *labelcfg.Set) {
	routerNames, hasExplicitRouters := labels.TCP.RouterNames()
	routerNames, hasExplicitRouters = b.claimObjectNames(workload, routerNames, hasExplicitRouters, b.tcpRouters, "TCP router")
	serviceNames, hasExplicitServices := labels.TCP.ServiceNames()
	serviceNames, hasExplicitServices = b.claimObjectNames(workload, serviceNames, hasExplicitServices, b.tcpServices, "TCP service")
	defaultName := defaultObjectName(names, labels)

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
		b.config.TCP.Services[serviceName] = buildTCPService(workload, labels.TCP.Services[serviceName], b.options)
	}
	for _, routerName := range routerNames {
		b.config.TCP.Routers[routerName] = buildTCPRouter(labels.TCP.Routers[routerName], b.defaultTCPServiceForRouter(workload, labels, routerName, serviceNames))
	}
}

func (b *configBuilder) addUDPWorkload(workload inventory.Workload, names GeneratedNames, labels *labelcfg.Set) {
	routerNames, hasExplicitRouters := labels.UDP.RouterNames()
	routerNames, hasExplicitRouters = b.claimObjectNames(workload, routerNames, hasExplicitRouters, b.udpRouters, "UDP router")
	serviceNames, hasExplicitServices := labels.UDP.ServiceNames()
	serviceNames, hasExplicitServices = b.claimObjectNames(workload, serviceNames, hasExplicitServices, b.udpServices, "UDP service")
	defaultName := defaultObjectName(names, labels)

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
		b.config.UDP.Services[serviceName] = buildUDPService(workload, labels.UDP.Services[serviceName], b.options)
	}
	for _, routerName := range routerNames {
		b.config.UDP.Routers[routerName] = buildUDPRouter(labels.UDP.Routers[routerName], b.defaultUDPServiceForRouter(workload, labels, routerName, serviceNames))
	}
}
