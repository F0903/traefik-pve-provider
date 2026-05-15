package traefik

import (
	"fmt"
	"strings"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	"github.com/traefik/genconf/dynamic"
)

func buildHTTPRouter(workload inventory.Workload, routerName, defaultService string, options Options) *dynamic.Router {
	prefix := "traefik.http.routers." + routerName + "."
	router := &dynamic.Router{
		Rule:    labelOrDefault(workload.TraefikLabels, prefix+"rule", defaultHTTPRule(routerName, options)),
		Service: labelOrDefault(workload.TraefikLabels, prefix+"service", defaultService),
	}

	if entrypoints := splitCSV(firstLabel(workload.TraefikLabels, prefix+"entrypoints", "traefik.entrypoints")); len(entrypoints) > 0 {
		router.EntryPoints = entrypoints
	} else if entrypoint := firstLabel(workload.TraefikLabels, prefix+"entrypoint", "traefik.entrypoint"); entrypoint != "" {
		router.EntryPoints = []string{entrypoint}
	}

	if middlewares := splitCSV(firstLabel(workload.TraefikLabels, prefix+"middlewares", "traefik.middlewares")); len(middlewares) > 0 {
		router.Middlewares = middlewares
	}

	if priority, ok := parseInt(workload.TraefikLabels[prefix+"priority"]); ok {
		router.Priority = priority
	}

	router.TLS = buildRouterTLS(workload.TraefikLabels, prefix)
	return router
}

func buildHTTPService(workload inventory.Workload, serviceName string) *dynamic.Service {
	prefix := "traefik.http.services." + serviceName + ".loadbalancer."
	passHostHeader := true
	if parsed, ok := parseBool(workload.TraefikLabels[prefix+"passhostheader"]); ok {
		passHostHeader = parsed
	}

	loadBalancer := &dynamic.ServersLoadBalancer{
		PassHostHeader: &passHostHeader,
		Servers:        buildHTTPServers(workload, serviceName),
	}

	if healthCheck := buildHealthCheck(workload.TraefikLabels, prefix); healthCheck != nil {
		loadBalancer.HealthCheck = healthCheck
	}
	if sticky := buildSticky(workload.TraefikLabels, prefix); sticky != nil {
		loadBalancer.Sticky = sticky
	}
	if flushInterval := strings.TrimSpace(workload.TraefikLabels[prefix+"responseforwarding.flushinterval"]); flushInterval != "" {
		loadBalancer.ResponseForwarding = &dynamic.ResponseForwarding{FlushInterval: flushInterval}
	}
	if serversTransport := firstLabel(workload.TraefikLabels, prefix+"serverstransport", "traefik.serverstransport"); serversTransport != "" {
		loadBalancer.ServersTransport = serversTransport
	}

	return &dynamic.Service{LoadBalancer: loadBalancer}
}

func buildHTTPServersTransport(labels map[string]string, transportName string) *dynamic.ServersTransport {
	prefix := "traefik.http.serverstransports." + transportName + "."
	transport := &dynamic.ServersTransport{
		ServerName:  strings.TrimSpace(labels[prefix+"servername"]),
		RootCAs:     splitCSV(labels[prefix+"rootcas"]),
		PeerCertURI: strings.TrimSpace(labels[prefix+"peercerturi"]),
	}

	if insecureSkipVerify, ok := parseBool(labels[prefix+"insecureskipverify"]); ok {
		transport.InsecureSkipVerify = insecureSkipVerify
	}
	if maxIdleConnsPerHost, ok := parseInt(labels[prefix+"maxidleconnsperhost"]); ok {
		transport.MaxIdleConnsPerHost = maxIdleConnsPerHost
	}
	if disableHTTP2, ok := parseBool(labels[prefix+"disablehttp2"]); ok {
		transport.DisableHTTP2 = disableHTTP2
	}
	if forwardingTimeouts := buildForwardingTimeouts(labels, prefix+"forwardingtimeouts."); forwardingTimeouts != nil {
		transport.ForwardingTimeouts = forwardingTimeouts
	}

	return transport
}

func buildInsecureHTTPServersTransport() *dynamic.ServersTransport {
	return &dynamic.ServersTransport{InsecureSkipVerify: true}
}

func buildForwardingTimeouts(labels map[string]string, prefix string) *dynamic.ForwardingTimeouts {
	timeouts := &dynamic.ForwardingTimeouts{
		DialTimeout:           strings.TrimSpace(labels[prefix+"dialtimeout"]),
		ResponseHeaderTimeout: strings.TrimSpace(labels[prefix+"responseheadertimeout"]),
		IdleConnTimeout:       strings.TrimSpace(labels[prefix+"idleconntimeout"]),
		ReadIdleTimeout:       strings.TrimSpace(labels[prefix+"readidletimeout"]),
		PingTimeout:           strings.TrimSpace(labels[prefix+"pingtimeout"]),
	}
	if timeouts.DialTimeout == "" &&
		timeouts.ResponseHeaderTimeout == "" &&
		timeouts.IdleConnTimeout == "" &&
		timeouts.ReadIdleTimeout == "" &&
		timeouts.PingTimeout == "" {
		return nil
	}
	return timeouts
}

func buildHTTPServers(workload inventory.Workload, serviceName string) []dynamic.Server {
	urlLabel := fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.url", serviceName)
	if url := strings.TrimSpace(workload.TraefikLabels[urlLabel]); url != "" {
		return []dynamic.Server{{URL: url}}
	}

	scheme, port := serviceSchemeAndPort(workload.TraefikLabels, serviceName)
	ipLabel := fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.ip", serviceName)
	if ip := strings.TrimSpace(workload.TraefikLabels[ipLabel]); ip != "" {
		return []dynamic.Server{{URL: serverURL(scheme, ip, port)}}
	}

	servers := make([]dynamic.Server, 0, len(workload.IPs))
	for _, ip := range workload.IPs {
		if strings.TrimSpace(ip.Address) == "" {
			continue
		}
		servers = append(servers, dynamic.Server{URL: serverURL(scheme, ip.Address, port)})
	}
	if len(servers) > 0 {
		return servers
	}

	return []dynamic.Server{{URL: serverURL(scheme, workload.Name+"."+workload.Node, port)}}
}

func serviceSchemeAndPort(labels map[string]string, serviceName string) (string, string) {
	prefix := "traefik.http.services." + serviceName + ".loadbalancer.server."
	scheme := firstLabel(labels, prefix+"scheme", "traefik.scheme")
	if scheme == "" {
		scheme = "http"
	}

	port := firstLabel(labels, prefix+"port", "traefik.port")
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return scheme, port
}

func httpServiceInsecureSkipVerify(labels map[string]string, serviceName string) (bool, bool) {
	serviceKey := "traefik.http.services." + serviceName + ".loadbalancer.insecureskipverify"
	if raw, exists := labels[serviceKey]; exists {
		return parseBool(raw)
	}
	if raw, exists := labels["traefik.insecureskipverify"]; exists {
		return parseBool(raw)
	}
	return false, false
}

func buildHealthCheck(labels map[string]string, servicePrefix string) *dynamic.ServerHealthCheck {
	path := strings.TrimSpace(labels[servicePrefix+"healthcheck.path"])
	if path == "" {
		return nil
	}

	healthCheck := &dynamic.ServerHealthCheck{Path: path}
	if interval := strings.TrimSpace(labels[servicePrefix+"healthcheck.interval"]); interval != "" {
		healthCheck.Interval = interval
	}
	if timeout := strings.TrimSpace(labels[servicePrefix+"healthcheck.timeout"]); timeout != "" {
		healthCheck.Timeout = timeout
	}
	if scheme := strings.TrimSpace(labels[servicePrefix+"healthcheck.scheme"]); scheme != "" {
		healthCheck.Scheme = scheme
	}
	if method := strings.TrimSpace(labels[servicePrefix+"healthcheck.method"]); method != "" {
		healthCheck.Method = method
	}
	if hostname := strings.TrimSpace(labels[servicePrefix+"healthcheck.hostname"]); hostname != "" {
		healthCheck.Hostname = hostname
	}
	if port, ok := parseInt(labels[servicePrefix+"healthcheck.port"]); ok {
		healthCheck.Port = port
	}
	if followRedirects, ok := parseBool(labels[servicePrefix+"healthcheck.followredirects"]); ok {
		healthCheck.FollowRedirects = &followRedirects
	}
	if headers := labelsWithPrefix(labels, servicePrefix+"healthcheck.headers."); len(headers) > 0 {
		healthCheck.Headers = headers
	}
	return healthCheck
}

func buildSticky(labels map[string]string, servicePrefix string) *dynamic.Sticky {
	cookiePrefix := servicePrefix + "sticky.cookie."
	name := strings.TrimSpace(labels[cookiePrefix+"name"])
	if name == "" {
		return nil
	}

	cookie := &dynamic.Cookie{Name: name}
	if secure, ok := parseBool(labels[cookiePrefix+"secure"]); ok {
		cookie.Secure = secure
	}
	if httpOnly, ok := parseBool(labels[cookiePrefix+"httponly"]); ok {
		cookie.HTTPOnly = httpOnly
	}
	if sameSite := strings.TrimSpace(labels[cookiePrefix+"samesite"]); sameSite != "" {
		cookie.SameSite = sameSite
	}

	return &dynamic.Sticky{Cookie: cookie}
}
