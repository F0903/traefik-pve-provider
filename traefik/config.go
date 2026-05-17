package traefik

import (
	"encoding/json"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	"github.com/traefik/genconf/dynamic"
	"github.com/traefik/genconf/dynamic/tls"
)

type Options struct {
	DefaultDomain string
}

type Result struct {
	Configuration *dynamic.Configuration
	Diagnostics   []Diagnostic
}

type configBuilder struct {
	options     Options
	config      *dynamic.Configuration
	diagnostics []Diagnostic

	httpRouters    map[string]objectOwner
	httpServices   map[string]objectOwner
	httpTransports map[string]objectOwner
	tcpRouters     map[string]objectOwner
	tcpServices    map[string]objectOwner
	udpRouters     map[string]objectOwner
	udpServices    map[string]objectOwner
}

func BuildConfiguration(snapshot inventory.Snapshot, options ...Options) *dynamic.Configuration {
	return Build(snapshot, firstOptions(options)).Configuration
}

func Build(snapshot inventory.Snapshot, options Options) Result {
	builder := newConfigBuilder(options)

	for _, workload := range snapshot.Workloads {
		labels := builder.validateLabels(workload)
		if !labels.Enabled() {
			continue
		}

		hasHTTP := labels.HasExplicitHTTP()
		hasTCP := labels.HasExplicitTCP()
		hasUDP := labels.HasExplicitUDP()

		if hasHTTP || (!hasTCP && !hasUDP) {
			builder.addHTTPWorkload(workload, labels)
		}
		if hasTCP {
			builder.addTCPWorkload(workload, labels)
		}
		if hasUDP {
			builder.addUDPWorkload(workload, labels)
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
