package labels

import (
	"github.com/F0903/traefik-pve-provider/traefik/ast"
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
)

func newLabelSet() *Set {
	set := &Set{}
	set.HTTP = newLabelProtocolSet(set, lexer.TokenHTTP)
	set.TCP = newLabelProtocolSet(set, lexer.TokenTCP)
	set.UDP = newLabelProtocolSet(set, lexer.TokenUDP)
	return set
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

func (s *Set) observeTokens(tokens []lexer.Token) {
	switch tokenTypeAt(tokens, 2) {
	case lexer.TokenHTTP:
		s.explicitHTTP = true
		s.HTTP.observeTokens(tokens)
	case lexer.TokenTCP:
		s.explicitTCP = true
		s.TCP.observeTokens(tokens)
	case lexer.TokenUDP:
		s.explicitUDP = true
		s.UDP.observeTokens(tokens)
	}
}

func (s *Set) Enabled() bool {
	value, ok := s.BoolValue(lexer.TokenEnable)
	return ok && value
}

func (s *Set) NameOverride() (string, bool) {
	return s.StringValue(lexer.TokenName)
}

func (s *Set) HasExplicitHTTP() bool {
	return s.explicitHTTP
}

func (s *Set) HasExplicitTCP() bool {
	return s.explicitTCP
}

func (s *Set) HasExplicitUDP() bool {
	return s.explicitUDP
}

func (s *Set) apply(node ast.Node, origin labelAssignmentOrigin) {
	switch n := node.(type) {
	case ast.Assignment:
		s.applyAssignment(n, origin)
	}
}

func (s *Set) applyAssignment(assignment ast.Assignment, origin labelAssignmentOrigin) {
	item := labelAssignment{
		assignment: assignment,
		origin:     origin,
	}
	s.assignments = append(s.assignments, item)
	s.indexAssignment(item)
}

func (s *Set) indexAssignment(assignment labelAssignment) {
	segments := assignment.assignment.Target.Segments()
	if len(segments) < 4 || segments[2].Type != lexer.TokenIdentifier || segments[2].Lexeme == "" {
		return
	}

	protocol := s.protocolSet(protocolForSegment(segments[0].Type))
	switch segments[1].Type {
	case lexer.TokenRouters:
		protocol.router(segments[2].Lexeme, assignment.origin)
	case lexer.TokenServices:
		protocol.service(segments[2].Lexeme, assignment.origin)
	case lexer.TokenServersTransports:
		protocol.serversTransport(segments[2].Lexeme, assignment.origin)
	}
}

func (s *Set) applyNameOverride(defaultName string) {
	name, ok := s.NameOverride()
	if !ok || name == defaultName {
		return
	}

	for index := range s.assignments {
		if s.assignments[index].origin != labelAssignmentOriginShorthand {
			continue
		}
		s.assignments[index].assignment.Target = renamedDefaultTarget(s.assignments[index].assignment.Target, defaultName, name)
	}
	s.rebuildResourceIndexes()
}

func (s *Set) rebuildResourceIndexes() {
	httpRouters := s.HTTP.explicitRouters
	httpServices := s.HTTP.explicitServices
	httpTransports := s.HTTP.explicitServersTransports
	tcpRouters := s.TCP.explicitRouters
	tcpServices := s.TCP.explicitServices
	tcpTransports := s.TCP.explicitServersTransports
	udpRouters := s.UDP.explicitRouters
	udpServices := s.UDP.explicitServices
	udpTransports := s.UDP.explicitServersTransports

	s.HTTP = newLabelProtocolSet(s, lexer.TokenHTTP)
	s.HTTP.explicitRouters = httpRouters
	s.HTTP.explicitServices = httpServices
	s.HTTP.explicitServersTransports = httpTransports
	s.TCP = newLabelProtocolSet(s, lexer.TokenTCP)
	s.TCP.explicitRouters = tcpRouters
	s.TCP.explicitServices = tcpServices
	s.TCP.explicitServersTransports = tcpTransports
	s.UDP = newLabelProtocolSet(s, lexer.TokenUDP)
	s.UDP.explicitRouters = udpRouters
	s.UDP.explicitServices = udpServices
	s.UDP.explicitServersTransports = udpTransports

	for _, assignment := range s.assignments {
		s.indexAssignment(assignment)
	}
}

func renamedDefaultTarget(target *ast.Target, from, to string) *ast.Target {
	segments := target.Segments()
	if len(segments) < 3 ||
		segments[2].Type != lexer.TokenIdentifier ||
		segments[2].Lexeme != from {
		return target
	}
	segments[2].Lexeme = to
	return ast.NewTarget(segments...)
}

func protocolForSegment(tokenType lexer.TokenType) labelProtocol {
	switch tokenType {
	case lexer.TokenTCP:
		return labelProtocolTCP
	case lexer.TokenUDP:
		return labelProtocolUDP
	default:
		return labelProtocolHTTP
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
		resource = &Resource{
			labels:     s.labels,
			protocol:   s.protocol,
			collection: collection,
			name:       name,
		}
		objects[name] = resource
	}
	if origin == labelAssignmentOriginExplicit {
		resource.explicit = true
	}
	return resource
}
