package traefik

import (
	"strconv"
	"strings"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
	"github.com/traefik/genconf/dynamic"
)

func buildHTTPRouter(source *labelcfg.Resource, routerName, defaultService string, options Options) *dynamic.Router {
	router := &dynamic.Router{
		Rule:    defaultHTTPRule(routerName, options),
		Service: defaultService,
	}
	if source != nil {
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
		router.TLS = buildRouterTLS(source)
	}
	return router
}

func buildHTTPService(workload inventory.Workload, source *labelcfg.Resource) *dynamic.Service {
	passHostHeader := true
	if parsed, ok := source.BoolValue(lexer.TokenLoadBalancer, lexer.TokenPassHostHeader); ok {
		passHostHeader = parsed
	}

	loadBalancer := &dynamic.ServersLoadBalancer{
		PassHostHeader: &passHostHeader,
		Servers:        buildHTTPServers(workload, source),
	}

	if healthCheck := buildHealthCheck(source); healthCheck != nil {
		loadBalancer.HealthCheck = healthCheck
	}
	if sticky := buildSticky(source); sticky != nil {
		loadBalancer.Sticky = sticky
	}
	if flushInterval, ok := source.StringValue(lexer.TokenLoadBalancer, lexer.TokenResponseForwarding, lexer.TokenFlushInterval); ok {
		loadBalancer.ResponseForwarding = &dynamic.ResponseForwarding{FlushInterval: flushInterval}
	}
	if serversTransport, ok := source.StringValue(lexer.TokenLoadBalancer, lexer.TokenServersTransport); ok {
		loadBalancer.ServersTransport = serversTransport
	}

	return &dynamic.Service{LoadBalancer: loadBalancer}
}

func buildHTTPServersTransport(source *labelcfg.Resource) *dynamic.ServersTransport {
	transport := &dynamic.ServersTransport{}

	if serverName, ok := source.StringValue(lexer.TokenServerName); ok {
		transport.ServerName = serverName
	}
	if rootCAs, ok := source.ListValue(lexer.TokenRootCAs); ok {
		transport.RootCAs = rootCAs
	}
	if peerCertURI, ok := source.StringValue(lexer.TokenPeerCertURI); ok {
		transport.PeerCertURI = peerCertURI
	}
	if insecureSkipVerify, ok := source.BoolValue(lexer.TokenInsecureSkipVerify); ok {
		transport.InsecureSkipVerify = insecureSkipVerify
	}
	if maxIdleConnsPerHost, ok := source.IntValue(lexer.TokenMaxIdleConnsPerHost); ok {
		transport.MaxIdleConnsPerHost = maxIdleConnsPerHost
	}
	if disableHTTP2, ok := source.BoolValue(lexer.TokenDisableHTTP2); ok {
		transport.DisableHTTP2 = disableHTTP2
	}
	if forwardingTimeouts := buildForwardingTimeouts(source); forwardingTimeouts != nil {
		transport.ForwardingTimeouts = forwardingTimeouts
	}

	return transport
}

func buildInsecureHTTPServersTransport() *dynamic.ServersTransport {
	return &dynamic.ServersTransport{InsecureSkipVerify: true}
}

func buildForwardingTimeouts(source *labelcfg.Resource) *dynamic.ForwardingTimeouts {
	timeouts := &dynamic.ForwardingTimeouts{}
	if dialTimeout, ok := source.StringValue(lexer.TokenForwardingTimeouts, lexer.TokenDialTimeout); ok {
		timeouts.DialTimeout = dialTimeout
	}
	if responseHeaderTimeout, ok := source.StringValue(lexer.TokenForwardingTimeouts, lexer.TokenResponseHeaderTimeout); ok {
		timeouts.ResponseHeaderTimeout = responseHeaderTimeout
	}
	if idleConnTimeout, ok := source.StringValue(lexer.TokenForwardingTimeouts, lexer.TokenIdleConnTimeout); ok {
		timeouts.IdleConnTimeout = idleConnTimeout
	}
	if readIdleTimeout, ok := source.StringValue(lexer.TokenForwardingTimeouts, lexer.TokenReadIdleTimeout); ok {
		timeouts.ReadIdleTimeout = readIdleTimeout
	}
	if pingTimeout, ok := source.StringValue(lexer.TokenForwardingTimeouts, lexer.TokenPingTimeout); ok {
		timeouts.PingTimeout = pingTimeout
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

func buildHTTPServers(workload inventory.Workload, source *labelcfg.Resource) []dynamic.Server {
	if url, ok := source.StringValue(lexer.TokenLoadBalancer, lexer.TokenServer, lexer.TokenURL); ok {
		return []dynamic.Server{{URL: url}}
	}

	scheme, port := serviceSchemeAndPort(source)
	if ip, ok := source.StringValue(lexer.TokenLoadBalancer, lexer.TokenServer, lexer.TokenIP); ok {
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

func serviceSchemeAndPort(source *labelcfg.Resource) (string, string) {
	scheme := "http"
	if parsed, ok := source.StringValue(lexer.TokenLoadBalancer, lexer.TokenServer, lexer.TokenScheme); ok {
		scheme = parsed
	}

	port := ""
	if parsed, ok := source.IntValue(lexer.TokenLoadBalancer, lexer.TokenServer, lexer.TokenPort); ok {
		port = strconv.Itoa(parsed)
	}
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return scheme, port
}

func httpServiceInsecureSkipVerify(source *labelcfg.Resource) (bool, bool) {
	return source.BoolValue(lexer.TokenLoadBalancer, lexer.TokenInsecureSkipVerify)
}

func buildHealthCheck(source *labelcfg.Resource) *dynamic.ServerHealthCheck {
	path, ok := source.StringValue(lexer.TokenLoadBalancer, lexer.TokenHealthCheck, lexer.TokenPath)
	if !ok {
		return nil
	}

	healthCheck := &dynamic.ServerHealthCheck{Path: path}
	if interval, ok := source.StringValue(lexer.TokenLoadBalancer, lexer.TokenHealthCheck, lexer.TokenInterval); ok {
		healthCheck.Interval = interval
	}
	if timeout, ok := source.StringValue(lexer.TokenLoadBalancer, lexer.TokenHealthCheck, lexer.TokenTimeout); ok {
		healthCheck.Timeout = timeout
	}
	if scheme, ok := source.StringValue(lexer.TokenLoadBalancer, lexer.TokenHealthCheck, lexer.TokenScheme); ok {
		healthCheck.Scheme = scheme
	}
	if method, ok := source.StringValue(lexer.TokenLoadBalancer, lexer.TokenHealthCheck, lexer.TokenMethod); ok {
		healthCheck.Method = method
	}
	if hostname, ok := source.StringValue(lexer.TokenLoadBalancer, lexer.TokenHealthCheck, lexer.TokenHostname); ok {
		healthCheck.Hostname = hostname
	}
	if port, ok := source.IntValue(lexer.TokenLoadBalancer, lexer.TokenHealthCheck, lexer.TokenPort); ok {
		healthCheck.Port = port
	}
	if followRedirects, ok := source.BoolValue(lexer.TokenLoadBalancer, lexer.TokenHealthCheck, lexer.TokenFollowRedirects); ok {
		healthCheck.FollowRedirects = &followRedirects
	}
	if headers := source.Headers(lexer.TokenLoadBalancer, lexer.TokenHealthCheck, lexer.TokenHeaders); len(headers) > 0 {
		healthCheck.Headers = headers
	}
	return healthCheck
}

func buildSticky(source *labelcfg.Resource) *dynamic.Sticky {
	name, ok := source.StringValue(lexer.TokenLoadBalancer, lexer.TokenSticky, lexer.TokenCookie, lexer.TokenName)
	if !ok {
		return nil
	}

	config := &dynamic.Cookie{Name: name}
	if secure, ok := source.BoolValue(lexer.TokenLoadBalancer, lexer.TokenSticky, lexer.TokenCookie, lexer.TokenSecure); ok {
		config.Secure = secure
	}
	if httpOnly, ok := source.BoolValue(lexer.TokenLoadBalancer, lexer.TokenSticky, lexer.TokenCookie, lexer.TokenHTTPOnly); ok {
		config.HTTPOnly = httpOnly
	}
	if sameSite, ok := source.StringValue(lexer.TokenLoadBalancer, lexer.TokenSticky, lexer.TokenCookie, lexer.TokenSameSite); ok {
		config.SameSite = sameSite
	}

	return &dynamic.Sticky{Cookie: config}
}
