package labels

import (
	"fmt"
	"sort"

	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
)

type labelProtocol int

const (
	labelProtocolHTTP labelProtocol = iota
	labelProtocolTCP
	labelProtocolUDP
)

type ProtocolSet struct {
	labels   *Set
	protocol lexer.TokenType

	Routers           map[string]*Resource
	Services          map[string]*Resource
	ServersTransports map[string]*Resource

	explicitRouters           bool
	explicitServices          bool
	explicitServersTransports bool
}

func newLabelProtocolSet(labels *Set, protocol lexer.TokenType) ProtocolSet {
	return ProtocolSet{
		labels:            labels,
		protocol:          protocol,
		Routers:           make(map[string]*Resource),
		Services:          make(map[string]*Resource),
		ServersTransports: make(map[string]*Resource),
	}
}

func protocolForSegment(tokenType lexer.TokenType) (labelProtocol, error) {
	switch tokenType {
	case lexer.TokenTCP:
		return labelProtocolTCP, nil
	case lexer.TokenUDP:
		return labelProtocolUDP, nil
	case lexer.TokenHTTP:
		return labelProtocolHTTP, nil
	default:
		return 0, fmt.Errorf("unsupported protocol token: %v", tokenType)
	}
}

func (s *Set) protocolSet(protocol labelProtocol) *ProtocolSet {
	switch protocol {
	case labelProtocolTCP:
		return &s.TCP
	case labelProtocolUDP:
		return &s.UDP
	default:
		return &s.HTTP
	}
}

func (s *ProtocolSet) observeTokens(tokens []lexer.Token) {
	if !isNamedProtocolObject(tokens) {
		return
	}

	switch tokenTypeAt(tokens, 4) {
	case lexer.TokenRouters:
		s.explicitRouters = true
	case lexer.TokenServices:
		s.explicitServices = true
	case lexer.TokenServersTransports:
		s.explicitServersTransports = true
	}
}

func (s ProtocolSet) RouterNames() ([]string, bool) {
	return sortedExplicitLabelResourceNames(s.Routers), s.explicitRouters
}

func (s ProtocolSet) ServiceNames() ([]string, bool) {
	return sortedExplicitLabelResourceNames(s.Services), s.explicitServices
}

func (s ProtocolSet) ServersTransportNames() ([]string, bool) {
	return sortedExplicitLabelResourceNames(s.ServersTransports), s.explicitServersTransports
}

func (s *ProtocolSet) router(name string, origin labelAssignmentOrigin) *Resource {
	return s.namedResource(s.Routers, lexer.TokenRouters, name, origin)
}

func (s *ProtocolSet) service(name string, origin labelAssignmentOrigin) *Resource {
	return s.namedResource(s.Services, lexer.TokenServices, name, origin)
}

func (s *ProtocolSet) serversTransport(name string, origin labelAssignmentOrigin) *Resource {
	return s.namedResource(s.ServersTransports, lexer.TokenServersTransports, name, origin)
}

func (s *ProtocolSet) namedResource(objects map[string]*Resource, collection lexer.TokenType, name string, origin labelAssignmentOrigin) *Resource {
	resource := objects[name]
	if resource == nil {
		resource = newResource(s.labels, s.protocol, collection, name)
		objects[name] = resource
	}
	if origin == labelAssignmentOriginExplicit {
		resource.explicit = true
	}
	return resource
}

func sortedExplicitLabelResourceNames(objects map[string]*Resource) []string {
	names := make([]string, 0, len(objects))
	for name, resource := range objects {
		if resource.explicit {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
