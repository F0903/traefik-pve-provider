package labels

import (
	"testing"

	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
)

func TestParseLabelSetTracksRootValuesAndExplicitProtocols(t *testing.T) {
	set, diagnostics := Parse(map[string]string{
		"traefik.enable":   "true",
		"traefik.port":     "8080",
		"traefik.tcp.port": "5432",
	}, "app")
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}

	if !set.Enabled() {
		t.Fatal("enabled = false")
	}
	if set.HasExplicitHTTP() {
		t.Fatal("explicit HTTP = true")
	}
	if !set.HasExplicitTCP() {
		t.Fatal("explicit TCP = false")
	}

	httpService := set.HTTP.Services["app"]
	if httpService == nil {
		t.Fatalf("missing default HTTP service: %#v", set.HTTP.Services)
	}
	if value, ok := httpService.IntValue(lexer.TokenLoadBalancer, lexer.TokenServer, lexer.TokenPort); !ok || value != 8080 {
		t.Fatalf("HTTP port = %#v, %t", value, ok)
	}
	if names, found := set.HTTP.ServiceNames(); len(names) != 0 || found {
		t.Fatalf("explicit HTTP service names = %#v, %t", names, found)
	}

	tcpService := set.TCP.Services["app"]
	if tcpService == nil {
		t.Fatalf("missing default TCP service: %#v", set.TCP.Services)
	}
	if value, ok := tcpService.IntValue(lexer.TokenLoadBalancer, lexer.TokenServer, lexer.TokenPort); !ok || value != 5432 {
		t.Fatalf("TCP port = %#v, %t", value, ok)
	}
	if names, found := set.TCP.ServiceNames(); len(names) != 0 || found {
		t.Fatalf("explicit TCP service names = %#v, %t", names, found)
	}
}

func TestParseLabelSetLetsExplicitLabelsOverrideShorthand(t *testing.T) {
	set, diagnostics := Parse(map[string]string{
		"traefik.port": "8080",
		"traefik.http.services.app.loadbalancer.server.port":        "9090",
		"traefik.http.services.app.loadbalancer.insecureskipverify": "true",
	}, "app")
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}

	service := set.HTTP.Services["app"]
	if service == nil {
		t.Fatalf("missing HTTP service: %#v", set.HTTP.Services)
	}
	if value, ok := service.IntValue(lexer.TokenLoadBalancer, lexer.TokenServer, lexer.TokenPort); !ok || value != 9090 {
		t.Fatalf("HTTP port = %#v, %t", value, ok)
	}
	if value, ok := service.BoolValue(lexer.TokenLoadBalancer, lexer.TokenInsecureSkipVerify); !ok || value != true {
		t.Fatalf("insecureSkipVerify = %#v, %t", value, ok)
	}
}

func TestParseLabelSetAppliesNameOverrideToShorthandAssignments(t *testing.T) {
	set, diagnostics := Parse(map[string]string{
		"traefik.name":        "opnsense",
		"traefik.port":        "443",
		"traefik.scheme":      "https",
		"traefik.tcp.port":    "8443",
		"traefik.entrypoints": "websecure",
	}, "firewall")
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}

	if set.HTTP.Services["firewall"] != nil {
		t.Fatalf("unexpected old default HTTP service: %#v", set.HTTP.Services["firewall"])
	}
	httpService := set.HTTP.Services["opnsense"]
	if httpService == nil {
		t.Fatalf("missing renamed HTTP service: %#v", set.HTTP.Services)
	}
	if value, ok := httpService.IntValue(lexer.TokenLoadBalancer, lexer.TokenServer, lexer.TokenPort); !ok || value != 443 {
		t.Fatalf("HTTP port = %#v, %t", value, ok)
	}
	if value, ok := httpService.StringValue(lexer.TokenLoadBalancer, lexer.TokenServer, lexer.TokenScheme); !ok || value != "https" {
		t.Fatalf("HTTP scheme = %#v, %t", value, ok)
	}

	httpRouter := set.HTTP.Routers["opnsense"]
	if httpRouter == nil {
		t.Fatalf("missing renamed HTTP router: %#v", set.HTTP.Routers)
	}
	if entrypoints, ok := httpRouter.ListValue(lexer.TokenEntryPoints); !ok || len(entrypoints) != 1 || entrypoints[0] != "websecure" {
		t.Fatalf("entrypoints = %#v, %t", entrypoints, ok)
	}

	tcpService := set.TCP.Services["opnsense"]
	if tcpService == nil {
		t.Fatalf("missing renamed TCP service: %#v", set.TCP.Services)
	}
	if value, ok := tcpService.IntValue(lexer.TokenLoadBalancer, lexer.TokenServer, lexer.TokenPort); !ok || value != 8443 {
		t.Fatalf("TCP port = %#v, %t", value, ok)
	}
}

func TestParseLabelSetLeavesExplicitAssignmentsOnTheirExplicitNames(t *testing.T) {
	set, diagnostics := Parse(map[string]string{
		"traefik.name": "opnsense",
		"traefik.port": "443",
		"traefik.http.services.firewall.loadbalancer.server.port": "8080",
	}, "firewall")
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}

	shorthandService := set.HTTP.Services["opnsense"]
	if shorthandService == nil {
		t.Fatalf("missing renamed shorthand service: %#v", set.HTTP.Services)
	}
	if value, ok := shorthandService.IntValue(lexer.TokenLoadBalancer, lexer.TokenServer, lexer.TokenPort); !ok || value != 443 {
		t.Fatalf("shorthand HTTP port = %#v, %t", value, ok)
	}

	explicitService := set.HTTP.Services["firewall"]
	if explicitService == nil {
		t.Fatalf("missing explicit service: %#v", set.HTTP.Services)
	}
	if value, ok := explicitService.IntValue(lexer.TokenLoadBalancer, lexer.TokenServer, lexer.TokenPort); !ok || value != 8080 {
		t.Fatalf("explicit HTTP port = %#v, %t", value, ok)
	}
}

func TestParseLabelSetCapturesNestedHeadersAndDomains(t *testing.T) {
	set, diagnostics := Parse(map[string]string{
		"traefik.http.routers.app.tls.domains[0].main":                           "example.com",
		"traefik.http.routers.app.tls.domains[0].sans":                           "*.example.com,api.example.com",
		"traefik.http.services.app.loadbalancer.healthcheck.headers.x-forwarded": "https",
	}, "app")
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}

	router := set.HTTP.Routers["app"]
	if router == nil {
		t.Fatalf("missing HTTP router: %#v", set.HTTP.Routers)
	}
	domains := router.TLSDomains(lexer.TokenTLS)
	if len(domains) != 1 || domains[0].Main != "example.com" {
		t.Fatalf("domains = %#v", domains)
	}
	if got := domains[0].SANs; len(got) != 2 || got[0] != "*.example.com" || got[1] != "api.example.com" {
		t.Fatalf("SANs = %#v", got)
	}

	service := set.HTTP.Services["app"]
	if service == nil {
		t.Fatalf("missing HTTP service: %#v", set.HTTP.Services)
	}
	headers := service.Headers(lexer.TokenLoadBalancer, lexer.TokenHealthCheck, lexer.TokenHeaders)
	if headers["x-forwarded"] != "https" {
		t.Fatalf("headers = %#v", headers)
	}
}

func TestParseLabelSetDiscoversKeywordObjectNames(t *testing.T) {
	set, diagnostics := Parse(map[string]string{
		"traefik.http.routers.secure.rule":                            "Host(`secure.example.com`)",
		"traefik.http.services.loadbalancer.loadbalancer.server.port": "8080",
	}, "app")
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}

	routerNames, foundRouters := set.HTTP.RouterNames()
	if !foundRouters || len(routerNames) != 1 || routerNames[0] != "secure" {
		t.Fatalf("router names = %#v, %t", routerNames, foundRouters)
	}

	serviceNames, foundServices := set.HTTP.ServiceNames()
	if !foundServices || len(serviceNames) != 1 || serviceNames[0] != "loadbalancer" {
		t.Fatalf("service names = %#v, %t", serviceNames, foundServices)
	}
}

func TestParseLabelSetMarksInvalidExplicitObjectGroups(t *testing.T) {
	set, diagnostics := Parse(map[string]string{
		"traefik.http.routers.my.app.rule": "Host(`app.example.com`)",
	}, "app")
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}

	routerNames, foundRouters := set.HTTP.RouterNames()
	if !foundRouters || len(routerNames) != 0 {
		t.Fatalf("router names = %#v, %t", routerNames, foundRouters)
	}
}
