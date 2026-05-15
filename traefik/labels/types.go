package labels

import (
	"github.com/F0903/traefik-pve-provider/traefik/ast"
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
	"github.com/F0903/traefik-pve-provider/traefik/ast/parser"
)

type labelProtocol int

const (
	labelProtocolHTTP labelProtocol = iota
	labelProtocolTCP
	labelProtocolUDP
)

type labelAssignmentOrigin int

const (
	labelAssignmentOriginShorthand labelAssignmentOrigin = iota
	labelAssignmentOriginExplicit
)

type Diagnostic struct {
	Key   string
	Value string
	Err   *parser.ParseError
}

type Set struct {
	assignments []labelAssignment

	explicitHTTP bool
	explicitTCP  bool
	explicitUDP  bool

	HTTP ProtocolSet
	TCP  ProtocolSet
	UDP  ProtocolSet
}

type labelAssignment struct {
	assignment ast.Assignment
	origin     labelAssignmentOrigin
}

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

type Resource struct {
	labels     *Set
	protocol   lexer.TokenType
	collection lexer.TokenType
	name       string
	explicit   bool
}

type TLSDomain struct {
	Main string
	SANs []string
}

type labelNamedValue struct {
	value  string
	origin labelAssignmentOrigin
}
