package traefik

import (
	"fmt"
	"strings"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
)

func (b *configBuilder) validateLabels(workload inventory.Workload) {
	for key, value := range workload.TraefikLabels {
		if !strings.HasPrefix(key, "traefik.") {
			continue
		}
		if !supportedConfigLabel(key) {
			b.addDiagnostic(workload, fmt.Sprintf("unsupported label %q ignored", key))
			continue
		}
		if booleanLabel(key) {
			if _, ok := parseBool(value); !ok {
				b.addDiagnostic(workload, fmt.Sprintf("label %q has invalid boolean value %q", key, value))
			}
		}
		if integerLabel(key) {
			if _, ok := parseInt(value); !ok {
				b.addDiagnostic(workload, fmt.Sprintf("label %q has invalid integer value %q", key, value))
			}
		}
	}
}

func supportedConfigLabel(key string) bool {
	switch key {
	case "traefik.enable",
		"traefik.name",
		"traefik.port",
		"traefik.scheme",
		"traefik.serverstransport",
		"traefik.insecureskipverify",
		"traefik.middlewares",
		"traefik.entrypoints",
		"traefik.entrypoint":
		return true
	}

	if option, ok := namedLabelOption(key, "traefik.http.routers."); ok {
		return supportedHTTPRouterOption(option)
	}
	if option, ok := namedLabelOption(key, "traefik.http.services."); ok {
		return supportedHTTPServiceLabelOption(option)
	}
	if option, ok := namedLabelOption(key, "traefik.http.serverstransports."); ok {
		return supportedHTTPServersTransportOption(option)
	}
	if option, ok := protocolLabelOption(key, "traefik.tcp."); ok {
		return supportedTCPShorthandOption(option)
	}
	if option, ok := namedLabelOption(key, "traefik.tcp.routers."); ok {
		return supportedTCPRouterOption(option)
	}
	if option, ok := namedLabelOption(key, "traefik.tcp.services."); ok {
		return supportedTCPServiceLabelOption(option)
	}
	if option, ok := protocolLabelOption(key, "traefik.udp."); ok {
		return supportedUDPShorthandOption(option)
	}
	if option, ok := namedLabelOption(key, "traefik.udp.routers."); ok {
		return supportedUDPRouterOption(option)
	}
	if option, ok := namedLabelOption(key, "traefik.udp.services."); ok {
		return supportedUDPServiceLabelOption(option)
	}

	return false
}

func protocolLabelOption(key, prefix string) (string, bool) {
	if !strings.HasPrefix(key, prefix) {
		return "", false
	}
	option := strings.TrimPrefix(key, prefix)
	if option == "" || strings.HasPrefix(option, "routers.") || strings.HasPrefix(option, "services.") {
		return "", false
	}
	return option, true
}

func namedLabelOption(key, prefix string) (string, bool) {
	if !strings.HasPrefix(key, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(key, prefix)
	name, option, ok := strings.Cut(rest, ".")
	if !ok || name == "" || option == "" {
		return "", false
	}
	return option, true
}

func supportedHTTPRouterOption(option string) bool {
	switch option {
	case "rule", "service", "entrypoints", "entrypoint", "middlewares", "priority", "tls", "tls.certresolver", "tls.options", "tls.domains":
		return true
	default:
		return tlsDomainLabelOption(option)
	}
}

func supportedHTTPServiceLabelOption(option string) bool {
	return strings.HasPrefix(option, "loadbalancer.") && supportedHTTPServiceOption(strings.TrimPrefix(option, "loadbalancer."))
}

func supportedHTTPServiceOption(option string) bool {
	switch option {
	case "server.url", "server.scheme", "server.port", "server.ip",
		"insecureskipverify",
		"passhostheader",
		"healthcheck.path", "healthcheck.interval", "healthcheck.timeout", "healthcheck.scheme", "healthcheck.method", "healthcheck.hostname", "healthcheck.port", "healthcheck.followredirects",
		"sticky.cookie.name", "sticky.cookie.secure", "sticky.cookie.httponly", "sticky.cookie.samesite",
		"responseforwarding.flushinterval",
		"serverstransport":
		return true
	default:
		return strings.HasPrefix(option, "healthcheck.headers.") && strings.TrimPrefix(option, "healthcheck.headers.") != ""
	}
}

func supportedHTTPServersTransportOption(option string) bool {
	switch option {
	case "servername", "insecureskipverify", "rootcas", "maxidleconnsperhost", "disablehttp2", "peercerturi",
		"forwardingtimeouts.dialtimeout",
		"forwardingtimeouts.responseheadertimeout",
		"forwardingtimeouts.idleconntimeout",
		"forwardingtimeouts.readidletimeout",
		"forwardingtimeouts.pingtimeout":
		return true
	default:
		return false
	}
}

func supportedTCPRouterOption(option string) bool {
	switch option {
	case "rule", "service", "entrypoints", "entrypoint", "middlewares", "priority", "tls", "tls.passthrough", "tls.certresolver", "tls.options", "tls.domains":
		return true
	default:
		return tlsDomainLabelOption(option)
	}
}

func supportedTCPShorthandOption(option string) bool {
	switch option {
	case "rule", "service", "entrypoints", "entrypoint", "middlewares", "priority",
		"tls", "tls.passthrough", "tls.certresolver", "tls.options", "tls.domains",
		"address", "ip", "port", "proxyprotocol.version", "terminationdelay":
		return true
	default:
		return tlsDomainLabelOption(option)
	}
}

func supportedTCPServiceLabelOption(option string) bool {
	return strings.HasPrefix(option, "loadbalancer.") && supportedTCPServiceOption(strings.TrimPrefix(option, "loadbalancer."))
}

func supportedTCPServiceOption(option string) bool {
	switch option {
	case "server.address", "server.port", "server.ip", "proxyprotocol.version", "terminationdelay":
		return true
	default:
		return false
	}
}

func supportedUDPRouterOption(option string) bool {
	switch option {
	case "entrypoints", "entrypoint", "service":
		return true
	default:
		return false
	}
}

func supportedUDPShorthandOption(option string) bool {
	switch option {
	case "entrypoints", "entrypoint", "service", "address", "ip", "port":
		return true
	default:
		return false
	}
}

func supportedUDPServiceLabelOption(option string) bool {
	return strings.HasPrefix(option, "loadbalancer.") && supportedUDPServiceOption(strings.TrimPrefix(option, "loadbalancer."))
}

func supportedUDPServiceOption(option string) bool {
	switch option {
	case "server.address", "server.port", "server.ip":
		return true
	default:
		return false
	}
}

func tlsDomainLabelOption(option string) bool {
	if !strings.HasPrefix(option, "tls.domains[") {
		return false
	}
	return tlsDomainLabelOptionPattern.MatchString(option)
}

func booleanLabel(key string) bool {
	return key == "traefik.enable" ||
		strings.HasSuffix(key, ".tls") ||
		strings.HasSuffix(key, ".tls.passthrough") ||
		strings.HasSuffix(key, ".passhostheader") ||
		strings.HasSuffix(key, ".healthcheck.followredirects") ||
		strings.HasSuffix(key, ".sticky.cookie.secure") ||
		strings.HasSuffix(key, ".sticky.cookie.httponly") ||
		strings.HasSuffix(key, ".insecureskipverify") ||
		strings.HasSuffix(key, ".disablehttp2")
}

func integerLabel(key string) bool {
	return key == "traefik.port" ||
		key == "traefik.tcp.port" ||
		key == "traefik.udp.port" ||
		strings.HasSuffix(key, ".priority") ||
		strings.HasSuffix(key, ".server.port") ||
		strings.HasSuffix(key, ".healthcheck.port") ||
		strings.HasSuffix(key, ".proxyprotocol.version") ||
		strings.HasSuffix(key, ".terminationdelay") ||
		strings.HasSuffix(key, ".maxidleconnsperhost")
}
