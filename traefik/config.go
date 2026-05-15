package traefik

import (
	"encoding/json"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	"github.com/traefik/genconf/dynamic"
	"github.com/traefik/genconf/dynamic/tls"
)

func BuildConfiguration(snapshot inventory.Snapshot, options ...Options) *dynamic.Configuration {
	return Build(snapshot, firstOptions(options)).Configuration
}

func Build(snapshot inventory.Snapshot, options Options) Result {
	builder := newConfigBuilder(options)

	for _, workload := range snapshot.Workloads {
		builder.validateLabels(workload)
		if !isEnabled(workload.TraefikLabels) {
			continue
		}

		hasHTTP := hasLabelPrefix(workload.TraefikLabels, "traefik.http.")
		hasTCP := hasLabelPrefix(workload.TraefikLabels, "traefik.tcp.")
		hasUDP := hasLabelPrefix(workload.TraefikLabels, "traefik.udp.")

		if hasHTTP || (!hasTCP && !hasUDP) {
			builder.addHTTPWorkload(workload)
		}
		if hasTCP {
			builder.addTCPWorkload(workload)
		}
		if hasUDP {
			builder.addUDPWorkload(workload)
		}
	}

	return Result{
		Configuration: builder.config,
		Diagnostics:   builder.diagnostics,
	}
}

func firstOptions(options []Options) Options {
	if len(options) == 0 {
		return Options{}
	}
	return options[0]
}

func newConfigBuilder(options Options) *configBuilder {
	return &configBuilder{
		options: options,
		config: &dynamic.Configuration{
			HTTP: &dynamic.HTTPConfiguration{
				Routers:           make(map[string]*dynamic.Router),
				Services:          make(map[string]*dynamic.Service),
				Middlewares:       make(map[string]*dynamic.Middleware),
				ServersTransports: make(map[string]*dynamic.ServersTransport),
			},
			TCP: &dynamic.TCPConfiguration{
				Routers:     make(map[string]*dynamic.TCPRouter),
				Services:    make(map[string]*dynamic.TCPService),
				Middlewares: make(map[string]*dynamic.TCPMiddleware),
			},
			UDP: &dynamic.UDPConfiguration{
				Routers:  make(map[string]*dynamic.UDPRouter),
				Services: make(map[string]*dynamic.UDPService),
			},
			TLS: &dynamic.TLSConfiguration{
				Options: make(map[string]tls.Options),
				Stores:  make(map[string]tls.Store),
			},
		},
		httpRouters:    make(map[string]objectOwner),
		httpServices:   make(map[string]objectOwner),
		httpTransports: make(map[string]objectOwner),
		tcpRouters:     make(map[string]objectOwner),
		tcpServices:    make(map[string]objectOwner),
		udpRouters:     make(map[string]objectOwner),
		udpServices:    make(map[string]objectOwner),
	}
}

func Marshal(configuration *dynamic.Configuration) ([]byte, error) {
	return json.Marshal(&dynamic.JSONPayload{Configuration: configuration})
}
