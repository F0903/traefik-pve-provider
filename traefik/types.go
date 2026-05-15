package traefik

import (
	"regexp"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	"github.com/traefik/genconf/dynamic"
)

type Options struct {
	DefaultDomain string
}

var tlsDomainLabelOptionPattern = regexp.MustCompile(`^tls\.domains\[\d+\]\.(main|sans)$`)

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
