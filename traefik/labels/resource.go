package labels

import "github.com/F0903/traefik-pve-provider/traefik/ast/lexer"

type Resource struct {
	labels     *Set
	protocol   lexer.TokenType
	collection lexer.TokenType
	name       string
	explicit   bool
	values     map[labelPathKey]indexedValue
	headers    map[labelPathKey]map[string]indexedString
	tlsDomains map[labelPathKey]map[int]*indexedTLSDomain
}

func newResource(labels *Set, protocol, collection lexer.TokenType, name string) *Resource {
	return &Resource{
		labels:     labels,
		protocol:   protocol,
		collection: collection,
		name:       name,
		values:     make(map[labelPathKey]indexedValue),
		headers:    make(map[labelPathKey]map[string]indexedString),
		tlsDomains: make(map[labelPathKey]map[int]*indexedTLSDomain),
	}
}
