package traefik

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
	"github.com/traefik/genconf/dynamic"
)

func labelsNote(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	builder.WriteString("```traefik\n")
	for _, key := range keys {
		labelKey := strings.TrimPrefix(key, "traefik.")
		builder.WriteString(labelKey)
		builder.WriteByte('=')
		builder.WriteString(labels[key])
		builder.WriteByte('\n')
	}
	builder.WriteString("```")
	return builder.String()
}

func TestBuildConfigurationIsJSONMarshalable(t *testing.T) {
	payload := BuildConfiguration(inventory.Snapshot{}, Options{})

	if _, err := json.Marshal(payload); err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
}

func TestMarshalConfigurationChangesWithBuiltConfig(t *testing.T) {
	first := BuildConfiguration(inventory.Snapshot{}, Options{})
	second := BuildConfiguration(inventory.Snapshot{}, Options{})
	second.HTTP.Routers["app"] = &dynamic.Router{
		Rule:    "Host(`app.example.com`)",
		Service: "app",
	}

	firstPayload, err := Marshal(first)
	if err != nil {
		t.Fatalf("marshalConfiguration(first) error = %v", err)
	}
	secondPayload, err := Marshal(second)
	if err != nil {
		t.Fatalf("marshalConfiguration(second) error = %v", err)
	}
	if string(firstPayload) == string(secondPayload) {
		t.Fatal("payloads matched for different configurations")
	}
}

func TestBuildConfigSkipsDisabledWorkloads(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:    100,
				Name:  "app",
				Node:  "pve1",
				Notes: labelsNote(map[string]string{"traefik.enable": "false"}),
			},
		},
	})

	if len(config.HTTP.Routers) != 0 {
		t.Fatalf("routers = %#v, want empty", config.HTTP.Routers)
	}
	if len(config.HTTP.Services) != 0 {
		t.Fatalf("services = %#v, want empty", config.HTTP.Services)
	}
}

func TestBuildConfigReusesParsedLabels(t *testing.T) {
	parsedLabels, parseDiagnostics := labelcfg.Parse(map[string]string{
		"traefik.enable": "true",
		"traefik.port":   "8080",
	}, "app")

	config := BuildPreparedConfiguration(PreparedSnapshot{
		Workloads: []PreparedWorkload{
			{
				Workload: inventory.Workload{
					ID:   100,
					Name: "app",
					Node: "pve1",
					IPs:  []inventory.IP{{Address: "10.0.0.10", Version: 4}},
				},
				Labels: LabelState{
					Raw:              map[string]string{"traefik.enable": "false"},
					Parsed:           parsedLabels,
					ParseDiagnostics: parseDiagnostics,
				},
			},
		},
	})

	if router := config.HTTP.Routers["app"]; router == nil {
		t.Fatalf("missing router from parsed labels: %#v", config.HTTP.Routers)
	}

	service := config.HTTP.Services["app"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("missing service from parsed labels: %#v", config.HTTP.Services)
	}
	if got := service.LoadBalancer.Servers[0].URL; got != "http://10.0.0.10:8080" {
		t.Fatalf("server URL = %q", got)
	}
}

func TestBuildConfigCreatesDefaultHTTPRouterAndService(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   100,
				Name: "app",
				Node: "pve1",
				IPs: []inventory.IP{
					{Address: "10.0.0.10", Version: 4},
					{Address: "10.0.0.11", Version: 4},
				},
				Notes: labelsNote(map[string]string{
					"traefik.enable": "true",
				}),
			},
		},
	})

	router := config.HTTP.Routers["app"]
	if router == nil {
		t.Fatalf("missing default router: %#v", config.HTTP.Routers)
	}
	if router.Rule != "Host(`app`)" {
		t.Fatalf("router rule = %q", router.Rule)
	}
	if router.Service != "app" {
		t.Fatalf("router service = %q", router.Service)
	}

	service := config.HTTP.Services["app"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("missing default service: %#v", config.HTTP.Services)
	}
	if len(service.LoadBalancer.Servers) != 2 {
		t.Fatalf("server count = %d", len(service.LoadBalancer.Servers))
	}
	if service.LoadBalancer.Servers[0].URL != "http://10.0.0.10:80" {
		t.Fatalf("first server URL = %q", service.LoadBalancer.Servers[0].URL)
	}
	if service.LoadBalancer.Servers[1].URL != "http://10.0.0.11:80" {
		t.Fatalf("second server URL = %q", service.LoadBalancer.Servers[1].URL)
	}
	if service.LoadBalancer.PassHostHeader == nil || !*service.LoadBalancer.PassHostHeader {
		t.Fatalf("PassHostHeader = %#v", service.LoadBalancer.PassHostHeader)
	}
}

func TestBuildConfigAppliesDefaultDomainAndShorthandLabels(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   101,
				Name: "traefik",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.20", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable":      "true",
					"traefik.port":        "8080",
					"traefik.middlewares": "local-only@file, compress-all@file",
				}),
			},
		},
	}, Options{DefaultDomain: "domain.net"})

	router := config.HTTP.Routers["traefik"]
	if router == nil {
		t.Fatalf("missing shorthand router: %#v", config.HTTP.Routers)
	}
	if router.Rule != "Host(`traefik.domain.net`)" {
		t.Fatalf("router rule = %q", router.Rule)
	}
	if router.Service != "traefik" {
		t.Fatalf("router service = %q", router.Service)
	}
	if got := router.Middlewares; len(got) != 2 || got[0] != "local-only@file" || got[1] != "compress-all@file" {
		t.Fatalf("middlewares = %#v", got)
	}

	service := config.HTTP.Services["traefik"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("missing shorthand service: %#v", config.HTTP.Services)
	}
	if got := service.LoadBalancer.Servers[0].URL; got != "http://10.0.0.20:8080" {
		t.Fatalf("server URL = %q", got)
	}
}

func TestBuildConfigNormalizesGeneratedWorkloadName(t *testing.T) {
	result := Build(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   101,
				Name: "My App_01",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.20", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable": "true",
				}),
			},
		},
	}, Options{DefaultDomain: "example.com"})

	router := result.Configuration.HTTP.Routers["my-app-01"]
	if router == nil {
		t.Fatalf("missing normalized router: %#v", result.Configuration.HTTP.Routers)
	}
	if router.Rule != "Host(`my-app-01.example.com`)" {
		t.Fatalf("router rule = %q", router.Rule)
	}
	if result.Configuration.HTTP.Services["my-app-01"] == nil {
		t.Fatalf("missing normalized service: %#v", result.Configuration.HTTP.Services)
	}
	if !diagnosticsContain(result.Diagnostics, `workload name "My App_01" normalized to "my-app-01"`) {
		t.Fatalf("missing name normalization diagnostic: %#v", result.Diagnostics)
	}
}

func TestBuildConfigAppliesNameOverrideAndHTTPSShorthand(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   102,
				Name: "firewall",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.1", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable":           "true",
					"traefik.name":             "opnsense",
					"traefik.scheme":           "https",
					"traefik.port":             "8443",
					"traefik.serverstransport": "ignore-ssl@file",
				}),
			},
		},
	}, Options{DefaultDomain: ".domain.net."})

	router := config.HTTP.Routers["opnsense"]
	if router == nil {
		t.Fatalf("missing override router: %#v", config.HTTP.Routers)
	}
	if router.Rule != "Host(`opnsense.domain.net`)" {
		t.Fatalf("router rule = %q", router.Rule)
	}

	service := config.HTTP.Services["opnsense"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("missing override service: %#v", config.HTTP.Services)
	}
	if got := service.LoadBalancer.Servers[0].URL; got != "https://10.0.0.1:8443" {
		t.Fatalf("server URL = %q", got)
	}
	if service.LoadBalancer.ServersTransport != "ignore-ssl@file" {
		t.Fatalf("servers transport = %q", service.LoadBalancer.ServersTransport)
	}
}

func TestBuildConfigNormalizesNameOverrideBeforeApplyingShorthands(t *testing.T) {
	result := Build(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   102,
				Name: "firewall",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.1", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable": "true",
					"traefik.name":   "OPNsense FW",
					"traefik.port":   "8443",
				}),
			},
		},
	}, Options{DefaultDomain: "domain.net"})

	router := result.Configuration.HTTP.Routers["opnsense-fw"]
	if router == nil {
		t.Fatalf("missing normalized override router: %#v", result.Configuration.HTTP.Routers)
	}
	if router.Rule != "Host(`opnsense-fw.domain.net`)" {
		t.Fatalf("router rule = %q", router.Rule)
	}
	if got := result.Configuration.HTTP.Services["opnsense-fw"].LoadBalancer.Servers[0].URL; got != "http://10.0.0.1:8443" {
		t.Fatalf("server URL = %q", got)
	}
	if !diagnosticsContain(result.Diagnostics, `label "traefik.name" value "OPNsense FW" normalized to "opnsense-fw"`) {
		t.Fatalf("missing override normalization diagnostic: %#v", result.Diagnostics)
	}
}

func TestBuildConfigRejectsUnusableNameOverride(t *testing.T) {
	result := Build(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   103,
				Name: "My App",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.1", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable": "true",
					"traefik.name":   "!!!",
				}),
			},
		},
	}, Options{})

	if result.Configuration.HTTP.Routers["my-app"] == nil {
		t.Fatalf("missing fallback router: %#v", result.Configuration.HTTP.Routers)
	}
	if result.Configuration.HTTP.Routers["!!!"] != nil {
		t.Fatalf("unexpected invalid override router: %#v", result.Configuration.HTTP.Routers)
	}
	if !diagnosticsContain(result.Diagnostics, `label "traefik.name" value "!!!" cannot be used as a generated Traefik name; using "my-app"`) {
		t.Fatalf("missing invalid override diagnostic: %#v", result.Diagnostics)
	}
}

func TestBuildConfigCreatesInsecureServersTransportFromShorthand(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   103,
				Name: "firewall",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.1", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable":             "true",
					"traefik.name":               "opnsense",
					"traefik.scheme":             "https",
					"traefik.port":               "443",
					"traefik.insecureskipverify": "true",
				}),
			},
		},
	})

	service := config.HTTP.Services["opnsense"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("missing service: %#v", config.HTTP.Services)
	}
	if got := service.LoadBalancer.Servers[0].URL; got != "https://10.0.0.1:443" {
		t.Fatalf("server URL = %q", got)
	}
	if service.LoadBalancer.ServersTransport != "opnsense-insecure" {
		t.Fatalf("servers transport = %q", service.LoadBalancer.ServersTransport)
	}

	transport := config.HTTP.ServersTransports["opnsense-insecure"]
	if transport == nil {
		t.Fatalf("missing generated servers transport: %#v", config.HTTP.ServersTransports)
	}
	if !transport.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify = false")
	}
}

func TestBuildConfigInsecureServersTransportRespectsExplicitTransport(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   104,
				Name: "app",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.2", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable":             "true",
					"traefik.scheme":             "https",
					"traefik.port":               "443",
					"traefik.serverstransport":   "ignore-ssl@file",
					"traefik.insecureskipverify": "true",
				}),
			},
		},
	})

	service := config.HTTP.Services["app"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("missing service: %#v", config.HTTP.Services)
	}
	if service.LoadBalancer.ServersTransport != "ignore-ssl@file" {
		t.Fatalf("servers transport = %q", service.LoadBalancer.ServersTransport)
	}
	if transport := config.HTTP.ServersTransports["app-insecure"]; transport != nil {
		t.Fatalf("generated transport with explicit transport = %#v", transport)
	}
}

func TestBuildConfigSupportsServiceScopedInsecureSkipVerify(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   105,
				Name: "app",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.3", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable": "true",
					"traefik.http.services.app.loadbalancer.server.scheme":      "https",
					"traefik.http.services.app.loadbalancer.server.port":        "8443",
					"traefik.http.services.app.loadbalancer.insecureskipverify": "true",
				}),
			},
		},
	})

	service := config.HTTP.Services["app"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("missing service: %#v", config.HTTP.Services)
	}
	if service.LoadBalancer.ServersTransport != "app-insecure" {
		t.Fatalf("servers transport = %q", service.LoadBalancer.ServersTransport)
	}
	if transport := config.HTTP.ServersTransports["app-insecure"]; transport == nil || !transport.InsecureSkipVerify {
		t.Fatalf("generated transport = %#v", transport)
	}
}

func TestBuildConfigFullLabelsOverrideShorthandLabels(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   106,
				Name: "app",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.30", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable": "true",
					"traefik.port":   "80",
					"traefik.scheme": "http",
					"traefik.http.services.app.loadbalancer.server.port":   "8080",
					"traefik.http.services.app.loadbalancer.server.scheme": "https",
				}),
			},
		},
	})

	service := config.HTTP.Services["app"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("missing service: %#v", config.HTTP.Services)
	}
	if got := service.LoadBalancer.Servers[0].URL; got != "https://10.0.0.30:8080" {
		t.Fatalf("server URL = %q", got)
	}
}

func TestBuildConfigLeavesExplicitObjectNamesUntouched(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   106,
				Name: "app",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.30", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable":                                               "true",
					"traefik.http.routers.my_router.rule":                          "Host(`app.example.com`)",
					"traefik.http.routers.my_router.service":                       "my_service",
					"traefik.http.services.my_service.loadbalancer.server.port":    "8080",
					"traefik.http.services.my_service.loadbalancer.server.scheme":  "https",
					"traefik.http.services.my_service.loadbalancer.passhostheader": "false",
				}),
			},
		},
	})

	if config.HTTP.Routers["my_router"] == nil {
		t.Fatalf("explicit router names were changed: %#v", config.HTTP.Routers)
	}
	service := config.HTTP.Services["my_service"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("explicit service names were changed: %#v", config.HTTP.Services)
	}
	if got := service.LoadBalancer.Servers[0].URL; got != "https://10.0.0.30:8080" {
		t.Fatalf("server URL = %q", got)
	}
}

func TestBuildConfigBindsMatchingRouterAndServiceNames(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   107,
				Name: "media",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.40", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable":                                          "true",
					"traefik.http.routers.jellyfin.rule":                      "Host(`jellyfin.example.com`)",
					"traefik.http.services.jellyfin.loadbalancer.server.port": "8096",
					"traefik.http.routers.plex.rule":                          "Host(`plex.example.com`)",
					"traefik.http.services.plex.loadbalancer.server.port":     "32400",
					"traefik.http.routers.sonarr.rule":                        "Host(`sonarr.example.com`)",
					"traefik.http.services.sonarr.loadbalancer.server.port":   "8989",
				}),
			},
		},
	})

	for routerName, serviceName := range map[string]string{
		"jellyfin": "jellyfin",
		"plex":     "plex",
		"sonarr":   "sonarr",
	} {
		if router := config.HTTP.Routers[routerName]; router == nil || router.Service != serviceName {
			t.Fatalf("router %s = %#v, want service %s", routerName, router, serviceName)
		}
	}
	if got := config.HTTP.Services["jellyfin"].LoadBalancer.Servers[0].URL; got != "http://10.0.0.40:8096" {
		t.Fatalf("jellyfin server = %q", got)
	}
	if got := config.HTTP.Services["plex"].LoadBalancer.Servers[0].URL; got != "http://10.0.0.40:32400" {
		t.Fatalf("plex server = %q", got)
	}
	if got := config.HTTP.Services["sonarr"].LoadBalancer.Servers[0].URL; got != "http://10.0.0.40:8989" {
		t.Fatalf("sonarr server = %q", got)
	}
}

func TestBuildConfigMergesDuplicateExplicitHTTPServiceServers(t *testing.T) {
	result := Build(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   105,
				Name: "app-a",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.41", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable":                                            "true",
					"traefik.http.routers.app.rule":                             "Host(`app.example.com`)",
					"traefik.http.routers.app.service":                          "app",
					"traefik.http.services.app.loadbalancer.server.port":        "8080",
					"traefik.http.services.app.loadbalancer.sticky.cookie.name": "session",
				}),
			},
			{
				ID:   106,
				Name: "app-b",
				Node: "pve2",
				IPs:  []inventory.IP{{Address: "10.0.0.42", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable":                                            "true",
					"traefik.http.routers.app.rule":                             "Host(`app.example.com`)",
					"traefik.http.routers.app.service":                          "app",
					"traefik.http.services.app.loadbalancer.server.port":        "8080",
					"traefik.http.services.app.loadbalancer.sticky.cookie.name": "session",
				}),
			},
		},
	}, Options{})

	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want empty", result.Diagnostics)
	}
	service := result.Configuration.HTTP.Services["app"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("missing merged service: %#v", result.Configuration.HTTP.Services)
	}
	if len(service.LoadBalancer.Servers) != 2 {
		t.Fatalf("server count = %d", len(service.LoadBalancer.Servers))
	}
	if service.LoadBalancer.Servers[0].URL != "http://10.0.0.41:8080" || service.LoadBalancer.Servers[1].URL != "http://10.0.0.42:8080" {
		t.Fatalf("servers = %#v", service.LoadBalancer.Servers)
	}
}

func TestBuildConfigSkipsDottedRouterAndServiceNames(t *testing.T) {
	result := Build(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   107,
				Name: "app",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.43", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable":                                        "true",
					"traefik.http.routers.my.app.rule":                      "Host(`app.example.com`)",
					"traefik.http.services.my.app.loadbalancer.server.port": "8080",
				}),
			},
		},
	}, Options{})

	if len(result.Configuration.HTTP.Routers) != 0 {
		t.Fatalf("routers = %#v, want empty", result.Configuration.HTTP.Routers)
	}
	if !diagnosticsContain(result.Diagnostics, `unsupported label "traefik.http.routers.my.app.rule" ignored`) {
		t.Fatalf("missing dotted router diagnostic: %#v", result.Diagnostics)
	}
	if !diagnosticsContain(result.Diagnostics, `no usable HTTP router names remain after collision checks`) {
		t.Fatalf("missing no-router diagnostic: %#v", result.Diagnostics)
	}
}

func TestBuildsUnsupportedAndInvalidLabels(t *testing.T) {
	result := Build(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   104,
				Name: "app",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.40", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable":                                       "true",
					"traefik.http.routers.app.priority":                    "high",
					"traefik.http.routers.app.unsupported":                 "value",
					"traefik.http.services.app.loadbalancer.server.port":   "eight",
					"traefik.http.services.app.loadbalancer.server.scheme": "http",
				}),
			},
		},
	}, Options{})

	if !diagnosticsContain(result.Diagnostics, `label "traefik.http.routers.app.priority" has invalid integer value "high"`) {
		t.Fatalf("missing priority diagnostic: %#v", result.Diagnostics)
	}
	if !diagnosticsContain(result.Diagnostics, `unsupported label "traefik.http.routers.app.unsupported" ignored`) {
		t.Fatalf("missing unsupported diagnostic: %#v", result.Diagnostics)
	}
	if !diagnosticsContain(result.Diagnostics, `label "traefik.http.services.app.loadbalancer.server.port" has invalid integer value "eight"`) {
		t.Fatalf("missing port diagnostic: %#v", result.Diagnostics)
	}
}

func TestBuildConfigDiagnosticsIncludeLabelSource(t *testing.T) {
	result := Build(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:    104,
				Name:  "app",
				Node:  "pve1",
				IPs:   []inventory.IP{{Address: "10.0.0.40", Version: 4}},
				Notes: "intro\n```traefik\nenable=true\nport=eight\n```",
			},
		},
	}, Options{})

	diagnostic := findDiagnostic(result.Diagnostics, `label "traefik.port" has invalid integer value "eight"`)
	if diagnostic == nil {
		t.Fatalf("missing port diagnostic: %#v", result.Diagnostics)
	}
	if diagnostic.Line != 4 || diagnostic.Fragment != "port=eight" {
		t.Fatalf("diagnostic source = %#v", diagnostic)
	}
}

func TestBuildConfigAvoidsDefaultNameCollisions(t *testing.T) {
	result := Build(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:    100,
				Name:  "app",
				Node:  "pve1",
				IPs:   []inventory.IP{{Address: "10.0.0.10", Version: 4}},
				Notes: labelsNote(map[string]string{"traefik.enable": "true", "traefik.port": "8080"}),
			},
			{
				ID:    101,
				Name:  "app",
				Node:  "pve2",
				IPs:   []inventory.IP{{Address: "10.0.0.11", Version: 4}},
				Notes: labelsNote(map[string]string{"traefik.enable": "true", "traefik.port": "8081"}),
			},
		},
	}, Options{DefaultDomain: "example.com"})

	if result.Configuration.HTTP.Routers["app"] == nil {
		t.Fatalf("missing first app router: %#v", result.Configuration.HTTP.Routers)
	}
	router := result.Configuration.HTTP.Routers["app-101"]
	if router == nil {
		t.Fatalf("missing collision-safe router: %#v", result.Configuration.HTTP.Routers)
	}
	if router.Rule != "Host(`app-101.example.com`)" {
		t.Fatalf("collision router rule = %q", router.Rule)
	}
	if service := result.Configuration.HTTP.Services["app-101"]; service == nil {
		t.Fatalf("missing collision-safe service: %#v", result.Configuration.HTTP.Services)
	}
	if !diagnosticsContain(result.Diagnostics, `HTTP router name "app" already exists; using "app-101"`) {
		t.Fatalf("missing router collision diagnostic: %#v", result.Diagnostics)
	}
	if !diagnosticsContain(result.Diagnostics, `HTTP service name "app" already exists; using "app-101"`) {
		t.Fatalf("missing service collision diagnostic: %#v", result.Diagnostics)
	}
}

func TestBuildConfigAppliesExplicitHTTPLabels(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   200,
				Name: "traefik",
				Node: "pve1",
				Notes: labelsNote(map[string]string{
					"traefik.enable":                                                                "true",
					"traefik.http.routers.dashboard.rule":                                           "Host(`traefik.site.net`)",
					"traefik.http.routers.dashboard.entrypoints":                                    "websecure, admin",
					"traefik.http.routers.dashboard.middlewares":                                    "local-only@file, compress-all@file",
					"traefik.http.routers.dashboard.priority":                                       "42",
					"traefik.http.routers.dashboard.service":                                        "dashboard",
					"traefik.http.services.dashboard.loadbalancer.server.port":                      "8080",
					"traefik.http.services.dashboard.loadbalancer.server.scheme":                    "https",
					"traefik.http.services.dashboard.loadbalancer.server.ip":                        "10.0.1.20",
					"traefik.http.services.dashboard.loadbalancer.passhostheader":                   "false",
					"traefik.http.services.dashboard.loadbalancer.healthcheck.path":                 "/health",
					"traefik.http.services.dashboard.loadbalancer.healthcheck.port":                 "8081",
					"traefik.http.services.dashboard.loadbalancer.sticky.cookie.name":               "session",
					"traefik.http.services.dashboard.loadbalancer.sticky.cookie.secure":             "true",
					"traefik.http.services.dashboard.loadbalancer.sticky.cookie.httponly":           "true",
					"traefik.http.services.dashboard.loadbalancer.responseforwarding.flushinterval": "100ms",
					"traefik.http.services.dashboard.loadbalancer.serverstransport":                 "insecure@file",
				}),
			},
		},
	})

	router := config.HTTP.Routers["dashboard"]
	if router == nil {
		t.Fatalf("missing router")
	}
	if router.Rule != "Host(`traefik.site.net`)" {
		t.Fatalf("rule = %q", router.Rule)
	}
	if router.Service != "dashboard" {
		t.Fatalf("service = %q", router.Service)
	}
	if router.Priority != 42 {
		t.Fatalf("priority = %d", router.Priority)
	}
	if got := router.EntryPoints; len(got) != 2 || got[0] != "websecure" || got[1] != "admin" {
		t.Fatalf("entrypoints = %#v", got)
	}
	if got := router.Middlewares; len(got) != 2 || got[0] != "local-only@file" || got[1] != "compress-all@file" {
		t.Fatalf("middlewares = %#v", got)
	}

	service := config.HTTP.Services["dashboard"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("missing service")
	}
	if got := service.LoadBalancer.Servers[0].URL; got != "https://10.0.1.20:8080" {
		t.Fatalf("server URL = %q", got)
	}
	if service.LoadBalancer.PassHostHeader == nil || *service.LoadBalancer.PassHostHeader {
		t.Fatalf("PassHostHeader = %#v", service.LoadBalancer.PassHostHeader)
	}
	if service.LoadBalancer.HealthCheck == nil || service.LoadBalancer.HealthCheck.Path != "/health" || service.LoadBalancer.HealthCheck.Port != 8081 {
		t.Fatalf("health check = %#v", service.LoadBalancer.HealthCheck)
	}
	if service.LoadBalancer.Sticky == nil || service.LoadBalancer.Sticky.Cookie == nil || service.LoadBalancer.Sticky.Cookie.Name != "session" {
		t.Fatalf("sticky = %#v", service.LoadBalancer.Sticky)
	}
	if !service.LoadBalancer.Sticky.Cookie.Secure || !service.LoadBalancer.Sticky.Cookie.HTTPOnly {
		t.Fatalf("sticky cookie = %#v", service.LoadBalancer.Sticky.Cookie)
	}
	if service.LoadBalancer.ResponseForwarding == nil || service.LoadBalancer.ResponseForwarding.FlushInterval != "100ms" {
		t.Fatalf("response forwarding = %#v", service.LoadBalancer.ResponseForwarding)
	}
	if service.LoadBalancer.ServersTransport != "insecure@file" {
		t.Fatalf("servers transport = %q", service.LoadBalancer.ServersTransport)
	}
}

func TestBuildConfigAppliesHealthCheckHeadersAndServersTransport(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   250,
				Name: "secure-backend",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.1.25", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable": "true",
					"traefik.http.services.app.loadbalancer.server.port":                           "8443",
					"traefik.http.services.app.loadbalancer.server.scheme":                         "https",
					"traefik.http.services.app.loadbalancer.serverstransport":                      "ignore-ssl",
					"traefik.http.services.app.loadbalancer.healthcheck.path":                      "/health",
					"traefik.http.services.app.loadbalancer.healthcheck.headers.x-forwarded-proto": "https",
					"traefik.http.serverstransports.ignore-ssl.insecureskipverify":                 "true",
					"traefik.http.serverstransports.ignore-ssl.maxidleconnsperhost":                "32",
					"traefik.http.serverstransports.ignore-ssl.forwardingtimeouts.dialtimeout":     "5s",
				}),
			},
		},
	})

	service := config.HTTP.Services["app"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("missing app service: %#v", config.HTTP.Services)
	}
	if got := service.LoadBalancer.Servers[0].URL; got != "https://10.0.1.25:8443" {
		t.Fatalf("server URL = %q", got)
	}
	if service.LoadBalancer.ServersTransport != "ignore-ssl" {
		t.Fatalf("servers transport = %q", service.LoadBalancer.ServersTransport)
	}
	if service.LoadBalancer.HealthCheck == nil || service.LoadBalancer.HealthCheck.Headers["x-forwarded-proto"] != "https" {
		t.Fatalf("health check = %#v", service.LoadBalancer.HealthCheck)
	}

	transport := config.HTTP.ServersTransports["ignore-ssl"]
	if transport == nil {
		t.Fatalf("missing servers transport: %#v", config.HTTP.ServersTransports)
	}
	if !transport.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify = false")
	}
	if transport.MaxIdleConnsPerHost != 32 {
		t.Fatalf("MaxIdleConnsPerHost = %d", transport.MaxIdleConnsPerHost)
	}
	if transport.ForwardingTimeouts == nil || transport.ForwardingTimeouts.DialTimeout != "5s" {
		t.Fatalf("forwarding timeouts = %#v", transport.ForwardingTimeouts)
	}
}

func TestBuildConfigAppliesRouterTLS(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   300,
				Name: "secure",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.0.30", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable":                                        "true",
					"traefik.http.routers.secure.rule":                      "Host(`secure.example.com`)",
					"traefik.http.routers.secure.tls":                       "true",
					"traefik.http.routers.secure.tls.certresolver":          "le",
					"traefik.http.routers.secure.tls.options":               "modern@file",
					"traefik.http.routers.secure.tls.domains[0].main":       "example.com",
					"traefik.http.routers.secure.tls.domains[0].sans":       "*.example.com,api.example.com",
					"traefik.http.routers.secure.tls.domains[1].main":       "example.net",
					"traefik.http.services.secure.loadbalancer.server.port": "8443",
				}),
			},
		},
	})

	router := config.HTTP.Routers["secure"]
	if router == nil || router.TLS == nil {
		t.Fatalf("router TLS = %#v", router)
	}
	if router.TLS.CertResolver != "le" || router.TLS.Options != "modern@file" {
		t.Fatalf("TLS config = %#v", router.TLS)
	}
	if len(router.TLS.Domains) != 2 {
		t.Fatalf("domains = %#v", router.TLS.Domains)
	}
	if router.TLS.Domains[0].Main != "example.com" {
		t.Fatalf("first domain = %#v", router.TLS.Domains[0])
	}
	if got := router.TLS.Domains[0].SANs; len(got) != 2 || got[0] != "*.example.com" || got[1] != "api.example.com" {
		t.Fatalf("first domain SANs = %#v", got)
	}
	if router.TLS.Domains[1].Main != "example.net" {
		t.Fatalf("second domain = %#v", router.TLS.Domains[1])
	}
}

func TestBuildConfigUsesURLAndHostnameFallback(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   400,
				Name: "url-app",
				Node: "pve1",
				Notes: labelsNote(map[string]string{
					"traefik.enable": "true",
					"traefik.http.services.url.loadbalancer.server.url": "http://upstream.local:9000",
					"traefik.http.routers.url.rule":                     "Host(`url.example.com`)",
					"traefik.http.routers.url.service":                  "url",
				}),
			},
			{
				ID:   401,
				Name: "fallback-app",
				Node: "pve2",
				Notes: labelsNote(map[string]string{
					"traefik.enable": "true",
				}),
			},
		},
	})

	if got := config.HTTP.Services["url"].LoadBalancer.Servers[0].URL; got != "http://upstream.local:9000" {
		t.Fatalf("explicit URL server = %q", got)
	}
	if got := config.HTTP.Services["fallback-app"].LoadBalancer.Servers[0].URL; got != "http://fallback-app.pve2:80" {
		t.Fatalf("fallback URL server = %q", got)
	}
}

func TestBuildConfigFromRealisticPrefixlessNotesSnapshot(t *testing.T) {
	traefikLabels := labelcfg.Extract("```traefik\nenable=true\nport=8080\nmiddlewares=local-only@file,compress-all@file\n```").Labels
	opnsenseLabels := labelcfg.Extract("```traefik\nenable=true\nname=opnsense\nscheme=https\nport=443\nserverstransport=ignore-ssl@file\n```").Labels

	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				Kind:  inventory.KindContainer,
				ID:    100,
				Name:  "traefik",
				Node:  "pve1",
				IPs:   []inventory.IP{{Address: "10.10.0.10", Version: 4}},
				Notes: labelsNote(traefikLabels),
			},
			{
				Kind:  inventory.KindVM,
				ID:    101,
				Name:  "firewall",
				Node:  "pve1",
				IPs:   []inventory.IP{{Address: "10.10.0.1", Version: 4}},
				Notes: labelsNote(opnsenseLabels),
			},
		},
	}, Options{DefaultDomain: "domain.net"})

	if got := config.HTTP.Routers["traefik"].Rule; got != "Host(`traefik.domain.net`)" {
		t.Fatalf("traefik rule = %q", got)
	}
	if got := config.HTTP.Services["traefik"].LoadBalancer.Servers[0].URL; got != "http://10.10.0.10:8080" {
		t.Fatalf("traefik server = %q", got)
	}
	if got := config.HTTP.Routers["opnsense"].Rule; got != "Host(`opnsense.domain.net`)" {
		t.Fatalf("opnsense rule = %q", got)
	}
	if got := config.HTTP.Services["opnsense"].LoadBalancer.Servers[0].URL; got != "https://10.10.0.1:443" {
		t.Fatalf("opnsense server = %q", got)
	}
	if got := config.HTTP.Services["opnsense"].LoadBalancer.ServersTransport; got != "ignore-ssl@file" {
		t.Fatalf("opnsense servers transport = %q", got)
	}
}

func TestBuildConfigAppliesTCPLabels(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   500,
				Name: "postgres",
				Node: "pve1",
				IPs: []inventory.IP{
					{Address: "10.0.2.10", Version: 4},
					{Address: "10.0.2.11", Version: 4},
				},
				Notes: labelsNote(map[string]string{
					"traefik.enable":                                             "true",
					"traefik.tcp.routers.pg.rule":                                "HostSNI(`pg.example.com`)",
					"traefik.tcp.routers.pg.entrypoints":                         "postgres",
					"traefik.tcp.routers.pg.service":                             "pg",
					"traefik.tcp.routers.pg.tls":                                 "true",
					"traefik.tcp.routers.pg.tls.passthrough":                     "true",
					"traefik.tcp.services.pg.loadbalancer.server.port":           "5432",
					"traefik.tcp.services.pg.loadbalancer.proxyprotocol.version": "2",
					"traefik.tcp.services.pg.loadbalancer.terminationdelay":      "100",
				}),
			},
		},
	})

	if len(config.HTTP.Routers) != 0 {
		t.Fatalf("HTTP routers = %#v, want none for TCP-only workload", config.HTTP.Routers)
	}

	router := config.TCP.Routers["pg"]
	if router == nil {
		t.Fatalf("missing TCP router")
	}
	if router.Rule != "HostSNI(`pg.example.com`)" || router.Service != "pg" {
		t.Fatalf("TCP router = %#v", router)
	}
	if got := router.EntryPoints; len(got) != 1 || got[0] != "postgres" {
		t.Fatalf("TCP entrypoints = %#v", got)
	}
	if router.TLS == nil || !router.TLS.Passthrough {
		t.Fatalf("TCP TLS = %#v", router.TLS)
	}

	service := config.TCP.Services["pg"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("missing TCP service")
	}
	if len(service.LoadBalancer.Servers) != 2 {
		t.Fatalf("TCP server count = %d", len(service.LoadBalancer.Servers))
	}
	if service.LoadBalancer.Servers[0].Address != "10.0.2.10:5432" {
		t.Fatalf("first TCP server = %q", service.LoadBalancer.Servers[0].Address)
	}
	if service.LoadBalancer.Servers[1].Address != "10.0.2.11:5432" {
		t.Fatalf("second TCP server = %q", service.LoadBalancer.Servers[1].Address)
	}
	if service.LoadBalancer.ProxyProtocol == nil || service.LoadBalancer.ProxyProtocol.Version != 2 {
		t.Fatalf("proxy protocol = %#v", service.LoadBalancer.ProxyProtocol)
	}
	if service.LoadBalancer.TerminationDelay == nil || *service.LoadBalancer.TerminationDelay != 100 {
		t.Fatalf("termination delay = %#v", service.LoadBalancer.TerminationDelay)
	}
}

func TestBuildConfigAppliesTCPShorthandLabels(t *testing.T) {
	labels := labelcfg.Extract("```traefik\nenable=true\ntcp.entrypoints=postgres\ntcp.rule=HostSNI(`pg.example.com`)\ntcp.port=5432\ntcp.tls=true\ntcp.tls.passthrough=true\ntcp.proxyprotocol.version=2\ntcp.terminationdelay=100\n```").Labels

	result := Build(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:    501,
				Name:  "pg",
				Node:  "pve1",
				IPs:   []inventory.IP{{Address: "10.0.2.10", Version: 4}},
				Notes: labelsNote(labels),
			},
		},
	}, Options{})

	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
	config := result.Configuration
	if len(config.HTTP.Routers) != 0 {
		t.Fatalf("HTTP routers = %#v, want none for TCP-only workload", config.HTTP.Routers)
	}

	router := config.TCP.Routers["pg"]
	if router == nil {
		t.Fatalf("missing TCP router")
	}
	if router.Rule != "HostSNI(`pg.example.com`)" || router.Service != "pg" {
		t.Fatalf("TCP router = %#v", router)
	}
	if got := router.EntryPoints; len(got) != 1 || got[0] != "postgres" {
		t.Fatalf("TCP entrypoints = %#v", got)
	}
	if router.TLS == nil || !router.TLS.Passthrough {
		t.Fatalf("TCP TLS = %#v", router.TLS)
	}

	service := config.TCP.Services["pg"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("missing TCP service")
	}
	if got := service.LoadBalancer.Servers[0].Address; got != "10.0.2.10:5432" {
		t.Fatalf("TCP server = %q", got)
	}
	if service.LoadBalancer.ProxyProtocol == nil || service.LoadBalancer.ProxyProtocol.Version != 2 {
		t.Fatalf("proxy protocol = %#v", service.LoadBalancer.ProxyProtocol)
	}
	if service.LoadBalancer.TerminationDelay == nil || *service.LoadBalancer.TerminationDelay != 100 {
		t.Fatalf("termination delay = %#v", service.LoadBalancer.TerminationDelay)
	}
}

func TestBuildConfigAppliesUDPLabels(t *testing.T) {
	config := BuildConfiguration(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				ID:   600,
				Name: "dns",
				Node: "pve1",
				IPs:  []inventory.IP{{Address: "10.0.3.10", Version: 4}},
				Notes: labelsNote(map[string]string{
					"traefik.enable":                                    "true",
					"traefik.udp.routers.dns.entrypoints":               "dns",
					"traefik.udp.routers.dns.service":                   "dns",
					"traefik.udp.services.dns.loadbalancer.server.port": "53",
				}),
			},
		},
	})

	if len(config.HTTP.Routers) != 0 {
		t.Fatalf("HTTP routers = %#v, want none for UDP-only workload", config.HTTP.Routers)
	}

	router := config.UDP.Routers["dns"]
	if router == nil {
		t.Fatalf("missing UDP router")
	}
	if router.Service != "dns" {
		t.Fatalf("UDP service = %q", router.Service)
	}
	if got := router.EntryPoints; len(got) != 1 || got[0] != "dns" {
		t.Fatalf("UDP entrypoints = %#v", got)
	}

	service := config.UDP.Services["dns"]
	if service == nil || service.LoadBalancer == nil {
		t.Fatalf("missing UDP service")
	}
	if got := service.LoadBalancer.Servers[0].Address; got != "10.0.3.10:53" {
		t.Fatalf("UDP server = %q", got)
	}
}

func diagnosticsContain(diagnostics []Diagnostic, fragment string) bool {
	return findDiagnostic(diagnostics, fragment) != nil
}

func findDiagnostic(diagnostics []Diagnostic, fragment string) *Diagnostic {
	for index := range diagnostics {
		if strings.Contains(diagnostics[index].Message, fragment) {
			return &diagnostics[index]
		}
	}
	return nil
}
