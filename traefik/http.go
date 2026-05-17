package traefik

import (
	"strconv"
	"strings"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
	"github.com/traefik/genconf/dynamic"
)

func buildHTTPRouter(source *labelcfg.Resource, routerName, defaultService string, options Options) *dynamic.Router {
	router := &dynamic.Router{
		Rule:    defaultHTTPRule(routerName, options),
		Service: defaultService,
	}
	if source != nil {
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
		router.TLS = buildRouterTLS(source)
	}
	return router
}

func buildHTTPService(workload inventory.Workload, source *labelcfg.Resource) *dynamic.Service {
	passHostHeader := true
	if parsed, ok := source.BoolValue("loadbalancer.passhostheader"); ok {
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
	if flushInterval, ok := source.StringValue("loadbalancer.responseforwarding.flushinterval"); ok {
		loadBalancer.ResponseForwarding = &dynamic.ResponseForwarding{FlushInterval: flushInterval}
	}
	if serversTransport, ok := source.StringValue("loadbalancer.serverstransport"); ok {
		loadBalancer.ServersTransport = serversTransport
	}

	return &dynamic.Service{LoadBalancer: loadBalancer}
}

func buildHTTPServersTransport(source *labelcfg.Resource) *dynamic.ServersTransport {
	transport := &dynamic.ServersTransport{}

	if serverName, ok := source.StringValue("servername"); ok {
		transport.ServerName = serverName
	}
	if rootCAs, ok := source.ListValue("rootcas"); ok {
		transport.RootCAs = rootCAs
	}
	if peerCertURI, ok := source.StringValue("peercerturi"); ok {
		transport.PeerCertURI = peerCertURI
	}
	if insecureSkipVerify, ok := source.BoolValue("insecureskipverify"); ok {
		transport.InsecureSkipVerify = insecureSkipVerify
	}
	if maxIdleConnsPerHost, ok := source.IntValue("maxidleconnsperhost"); ok {
		transport.MaxIdleConnsPerHost = maxIdleConnsPerHost
	}
	if disableHTTP2, ok := source.BoolValue("disablehttp2"); ok {
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
	if dialTimeout, ok := source.StringValue("forwardingtimeouts.dialtimeout"); ok {
		timeouts.DialTimeout = dialTimeout
	}
	if responseHeaderTimeout, ok := source.StringValue("forwardingtimeouts.responseheadertimeout"); ok {
		timeouts.ResponseHeaderTimeout = responseHeaderTimeout
	}
	if idleConnTimeout, ok := source.StringValue("forwardingtimeouts.idleconntimeout"); ok {
		timeouts.IdleConnTimeout = idleConnTimeout
	}
	if readIdleTimeout, ok := source.StringValue("forwardingtimeouts.readidletimeout"); ok {
		timeouts.ReadIdleTimeout = readIdleTimeout
	}
	if pingTimeout, ok := source.StringValue("forwardingtimeouts.pingtimeout"); ok {
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
	if url, ok := source.StringValue("loadbalancer.server.url"); ok {
		return []dynamic.Server{{URL: url}}
	}

	scheme, port := serviceSchemeAndPort(source)
	if ip, ok := source.StringValue("loadbalancer.server.ip"); ok {
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
	if parsed, ok := source.StringValue("loadbalancer.server.scheme"); ok {
		scheme = parsed
	}

	port := ""
	if parsed, ok := source.IntValue("loadbalancer.server.port"); ok {
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
	return source.BoolValue("loadbalancer.insecureskipverify")
}

func buildHealthCheck(source *labelcfg.Resource) *dynamic.ServerHealthCheck {
	path, ok := source.StringValue("loadbalancer.healthcheck.path")
	if !ok {
		return nil
	}

	healthCheck := &dynamic.ServerHealthCheck{Path: path}
	if interval, ok := source.StringValue("loadbalancer.healthcheck.interval"); ok {
		healthCheck.Interval = interval
	}
	if timeout, ok := source.StringValue("loadbalancer.healthcheck.timeout"); ok {
		healthCheck.Timeout = timeout
	}
	if scheme, ok := source.StringValue("loadbalancer.healthcheck.scheme"); ok {
		healthCheck.Scheme = scheme
	}
	if method, ok := source.StringValue("loadbalancer.healthcheck.method"); ok {
		healthCheck.Method = method
	}
	if hostname, ok := source.StringValue("loadbalancer.healthcheck.hostname"); ok {
		healthCheck.Hostname = hostname
	}
	if port, ok := source.IntValue("loadbalancer.healthcheck.port"); ok {
		healthCheck.Port = port
	}
	if followRedirects, ok := source.BoolValue("loadbalancer.healthcheck.followredirects"); ok {
		healthCheck.FollowRedirects = &followRedirects
	}
	if headers := source.Headers("loadbalancer.healthcheck.headers"); len(headers) > 0 {
		healthCheck.Headers = headers
	}
	return healthCheck
}

func buildSticky(source *labelcfg.Resource) *dynamic.Sticky {
	name, ok := source.StringValue("loadbalancer.sticky.cookie.name")
	if !ok {
		return nil
	}

	config := &dynamic.Cookie{Name: name}
	if secure, ok := source.BoolValue("loadbalancer.sticky.cookie.secure"); ok {
		config.Secure = secure
	}
	if httpOnly, ok := source.BoolValue("loadbalancer.sticky.cookie.httponly"); ok {
		config.HTTPOnly = httpOnly
	}
	if sameSite, ok := source.StringValue("loadbalancer.sticky.cookie.samesite"); ok {
		config.SameSite = sameSite
	}

	return &dynamic.Sticky{Cookie: config}
}
