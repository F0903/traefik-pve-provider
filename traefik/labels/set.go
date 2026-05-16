package labels

import (
	"github.com/F0903/traefik-pve-provider/traefik/ast"
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
)

type Set struct {
	assignments []labelAssignment
	values      map[labelPathKey]indexedValue

	explicitHTTP bool
	explicitTCP  bool
	explicitUDP  bool

	HTTP ProtocolSet
	TCP  ProtocolSet
	UDP  ProtocolSet
}

func newLabelSet() *Set {
	set := &Set{
		values: make(map[labelPathKey]indexedValue),
	}
	set.HTTP = newLabelProtocolSet(set, lexer.TokenHTTP)
	set.TCP = newLabelProtocolSet(set, lexer.TokenTCP)
	set.UDP = newLabelProtocolSet(set, lexer.TokenUDP)
	return set
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
	value, hasValue := assignmentValue(assignment.assignment.Value)
	if len(segments) == 1 {
		if hasValue {
			putIndexedValue(s.values, pathKeyForSegments(segments), value, assignment.origin)
		}
		return
	}

	resource := s.resourceForSegments(segments, assignment.origin)
	if resource == nil {
		return
	}
	if !hasValue {
		return
	}

	rest := segments[3:]
	putIndexedValue(resource.values, pathKeyForSegments(rest), value, assignment.origin)
	resource.indexHeader(rest, value, assignment.origin)
	resource.indexTLSDomain(rest, value, assignment.origin)
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
	s.values = make(map[labelPathKey]indexedValue)

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

func (s *Set) resourceForSegments(segments []ast.Segment, origin labelAssignmentOrigin) *Resource {
	if len(segments) < 4 || segments[2].Type != lexer.TokenIdentifier || segments[2].Lexeme == "" {
		return nil
	}

	prot, err := protocolForSegment(segments[0].Type)
	if err != nil {
		return nil
	}
	protocol := s.protocolSet(prot)
	switch segments[1].Type {
	case lexer.TokenRouters:
		return protocol.router(segments[2].Lexeme, origin)
	case lexer.TokenServices:
		return protocol.service(segments[2].Lexeme, origin)
	case lexer.TokenServersTransports:
		return protocol.serversTransport(segments[2].Lexeme, origin)
	default:
		return nil
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
