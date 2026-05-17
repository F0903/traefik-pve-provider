package schema

import "slices"

const (
	RootEnable = "enable"
	RootName   = "name"
)

// Schema rows are label path patterns. Plain rows register their own path as
// the target, while rows with "source -> target" register shorthand aliases.
//
// Examples:
//
//	http.routers.{name}.rule
//	entrypoint -> http.routers.{default}.entrypoint
//
// For resource targets, the path after the resource-name capture becomes the
// resource lookup key. For example,
// http.services.{name}.loadbalancer.server.port stores the value under the
// resource key loadbalancer.server.port.
//
// Capture syntax is generic. The only special capture is {default}, which
// expands to the workload default name in shorthand targets instead of reading
// from the source path.
//
//	{capture}             captures one path segment under the name "capture".
//	domains[{capture}]    captures the numeric TLS domain index under the name "capture".
var rows = func() []Spec {
	schema := []Spec{
		label(RootEnable, ValueBool),
		label(RootName, ValueString),
	}

	schema = append(schema, httpSchema()...)
	schema = append(schema, tcpSchema()...)
	schema = append(schema, udpSchema()...)
	return schema
}()

// Rows returns a copy of the compiled label schema rows.
func Rows() []Spec {
	return slices.Clone(rows)
}

func httpSchema() []Spec {
	return []Spec{
		// HTTP shorthand labels. These expand onto the default HTTP router/service.
		label("port -> http.services.{default}.loadbalancer.server.port", ValueInt),
		label("scheme -> http.services.{default}.loadbalancer.server.scheme", ValueString),
		label("insecureskipverify -> http.services.{default}.loadbalancer.insecureskipverify", ValueBool),
		label("serverstransport -> http.services.{default}.loadbalancer.serverstransport", ValueString),
		label("middlewares -> http.routers.{default}.middlewares", ValueCSV),
		label("entrypoints -> http.routers.{default}.entrypoints", ValueCSV),
		label("entrypoint -> http.routers.{default}.entrypoint", ValueString),

		// HTTP routers.
		label("http.routers.{name}.rule", ValueString),
		label("http.routers.{name}.service", ValueString),
		label("http.routers.{name}.entrypoints", ValueCSV),
		label("http.routers.{name}.entrypoint", ValueString),
		label("http.routers.{name}.middlewares", ValueCSV),
		label("http.routers.{name}.priority", ValueInt),

		// HTTP router TLS.
		label("http.routers.{name}.tls", ValueBool),
		label("http.routers.{name}.tls.certresolver", ValueString),
		label("http.routers.{name}.tls.options", ValueString),
		label("http.routers.{name}.tls.domains", ValueCSV),
		label("http.routers.{name}.tls.domains[{domain}]", ValueCSV),
		label("http.routers.{name}.tls.domains[{domain}].main", ValueString),
		label("http.routers.{name}.tls.domains[{domain}].sans", ValueCSV),

		// HTTP services.
		label("http.services.{name}.loadbalancer.server.url", ValueString),
		label("http.services.{name}.loadbalancer.server.scheme", ValueString),
		label("http.services.{name}.loadbalancer.server.port", ValueInt),
		label("http.services.{name}.loadbalancer.server.ip", ValueString),
		label("http.services.{name}.loadbalancer.insecureskipverify", ValueBool),
		label("http.services.{name}.loadbalancer.passhostheader", ValueBool),
		label("http.services.{name}.loadbalancer.serverstransport", ValueString),

		// HTTP health checks.
		label("http.services.{name}.loadbalancer.healthcheck.path", ValueString),
		label("http.services.{name}.loadbalancer.healthcheck.interval", ValueString),
		label("http.services.{name}.loadbalancer.healthcheck.timeout", ValueString),
		label("http.services.{name}.loadbalancer.healthcheck.scheme", ValueString),
		label("http.services.{name}.loadbalancer.healthcheck.method", ValueString),
		label("http.services.{name}.loadbalancer.healthcheck.hostname", ValueString),
		label("http.services.{name}.loadbalancer.healthcheck.port", ValueInt),
		label("http.services.{name}.loadbalancer.healthcheck.followredirects", ValueBool),
		label("http.services.{name}.loadbalancer.healthcheck.headers.{header}", ValueString),

		// HTTP sticky cookies and response forwarding.
		label("http.services.{name}.loadbalancer.sticky.cookie.name", ValueString),
		label("http.services.{name}.loadbalancer.sticky.cookie.secure", ValueBool),
		label("http.services.{name}.loadbalancer.sticky.cookie.httponly", ValueBool),
		label("http.services.{name}.loadbalancer.sticky.cookie.samesite", ValueString),
		label("http.services.{name}.loadbalancer.responseforwarding.flushinterval", ValueString),

		// HTTP servers transports.
		label("http.serverstransports.{name}.servername", ValueString),
		label("http.serverstransports.{name}.insecureskipverify", ValueBool),
		label("http.serverstransports.{name}.rootcas", ValueCSV),
		label("http.serverstransports.{name}.maxidleconnsperhost", ValueInt),
		label("http.serverstransports.{name}.disablehttp2", ValueBool),
		label("http.serverstransports.{name}.peercerturi", ValueString),
		label("http.serverstransports.{name}.forwardingtimeouts.dialtimeout", ValueString),
		label("http.serverstransports.{name}.forwardingtimeouts.responseheadertimeout", ValueString),
		label("http.serverstransports.{name}.forwardingtimeouts.idleconntimeout", ValueString),
		label("http.serverstransports.{name}.forwardingtimeouts.readidletimeout", ValueString),
		label("http.serverstransports.{name}.forwardingtimeouts.pingtimeout", ValueString),
	}
}

func tcpSchema() []Spec {
	return []Spec{
		// TCP shorthand router labels.
		label("tcp.rule -> tcp.routers.{default}.rule", ValueString),
		label("tcp.service -> tcp.routers.{default}.service", ValueString),
		label("tcp.entrypoints -> tcp.routers.{default}.entrypoints", ValueCSV),
		label("tcp.entrypoint -> tcp.routers.{default}.entrypoint", ValueString),
		label("tcp.middlewares -> tcp.routers.{default}.middlewares", ValueCSV),
		label("tcp.priority -> tcp.routers.{default}.priority", ValueInt),
		label("tcp.tls -> tcp.routers.{default}.tls", ValueBool),
		label("tcp.tls.passthrough -> tcp.routers.{default}.tls.passthrough", ValueBool),
		label("tcp.tls.certresolver -> tcp.routers.{default}.tls.certresolver", ValueString),
		label("tcp.tls.options -> tcp.routers.{default}.tls.options", ValueString),
		label("tcp.tls.domains -> tcp.routers.{default}.tls.domains", ValueCSV),
		label("tcp.tls.domains[{domain}] -> tcp.routers.{default}.tls.domains[{domain}]", ValueCSV),
		label("tcp.tls.domains[{domain}].main -> tcp.routers.{default}.tls.domains[{domain}].main", ValueString),
		label("tcp.tls.domains[{domain}].sans -> tcp.routers.{default}.tls.domains[{domain}].sans", ValueCSV),

		// TCP shorthand service labels.
		label("tcp.address -> tcp.services.{default}.loadbalancer.server.address", ValueString),
		label("tcp.ip -> tcp.services.{default}.loadbalancer.server.ip", ValueString),
		label("tcp.port -> tcp.services.{default}.loadbalancer.server.port", ValueInt),
		label("tcp.proxyprotocol.version -> tcp.services.{default}.loadbalancer.proxyprotocol.version", ValueInt),
		label("tcp.terminationdelay -> tcp.services.{default}.loadbalancer.terminationdelay", ValueInt),

		// TCP routers.
		label("tcp.routers.{name}.rule", ValueString),
		label("tcp.routers.{name}.service", ValueString),
		label("tcp.routers.{name}.entrypoints", ValueCSV),
		label("tcp.routers.{name}.entrypoint", ValueString),
		label("tcp.routers.{name}.middlewares", ValueCSV),
		label("tcp.routers.{name}.priority", ValueInt),

		// TCP router TLS.
		label("tcp.routers.{name}.tls", ValueBool),
		label("tcp.routers.{name}.tls.passthrough", ValueBool),
		label("tcp.routers.{name}.tls.certresolver", ValueString),
		label("tcp.routers.{name}.tls.options", ValueString),
		label("tcp.routers.{name}.tls.domains", ValueCSV),
		label("tcp.routers.{name}.tls.domains[{domain}]", ValueCSV),
		label("tcp.routers.{name}.tls.domains[{domain}].main", ValueString),
		label("tcp.routers.{name}.tls.domains[{domain}].sans", ValueCSV),

		// TCP services.
		label("tcp.services.{name}.loadbalancer.server.port", ValueInt),
		label("tcp.services.{name}.loadbalancer.server.ip", ValueString),
		label("tcp.services.{name}.loadbalancer.server.address", ValueString),
		label("tcp.services.{name}.loadbalancer.proxyprotocol.version", ValueInt),
		label("tcp.services.{name}.loadbalancer.terminationdelay", ValueInt),
	}
}

func udpSchema() []Spec {
	return []Spec{
		// UDP shorthand router labels.
		label("udp.service -> udp.routers.{default}.service", ValueString),
		label("udp.entrypoints -> udp.routers.{default}.entrypoints", ValueCSV),
		label("udp.entrypoint -> udp.routers.{default}.entrypoint", ValueString),

		// UDP shorthand service labels.
		label("udp.address -> udp.services.{default}.loadbalancer.server.address", ValueString),
		label("udp.ip -> udp.services.{default}.loadbalancer.server.ip", ValueString),
		label("udp.port -> udp.services.{default}.loadbalancer.server.port", ValueInt),

		// UDP routers.
		label("udp.routers.{name}.service", ValueString),
		label("udp.routers.{name}.entrypoints", ValueCSV),
		label("udp.routers.{name}.entrypoint", ValueString),

		// UDP services.
		label("udp.services.{name}.loadbalancer.server.port", ValueInt),
		label("udp.services.{name}.loadbalancer.server.ip", ValueString),
		label("udp.services.{name}.loadbalancer.server.address", ValueString),
	}
}
