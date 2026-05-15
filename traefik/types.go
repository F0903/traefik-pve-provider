package traefik

import (
	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	"github.com/traefik/genconf/dynamic"
)

type Options struct {
	DefaultDomain string
}

type Result struct {
	Configuration *dynamic.Configuration
	Diagnostics   []Diagnostic
}

type Diagnostic struct {
	Node    string
	Kind    inventory.Kind
	ID      int
	Message string
}

func (b *configBuilder) addDiagnostic(workload inventory.Workload, message string) {
	b.diagnostics = append(b.diagnostics, Diagnostic{
		Node:    workload.Node,
		Kind:    workload.Kind,
		ID:      workload.ID,
		Message: message,
	})
}

type objectOwner struct {
	Node string
	Kind inventory.Kind
	ID   int
	Name string
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
