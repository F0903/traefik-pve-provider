package labels

import (
	"strconv"
	"strings"

	"github.com/F0903/traefik-pve-provider/traefik/ast"
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
)

type labelPathKey string

type indexedValue struct {
	value  any
	origin labelAssignmentOrigin
}

type indexedString struct {
	value  string
	origin labelAssignmentOrigin
}

type indexedTLSDomain struct {
	main       string
	mainOrigin labelAssignmentOrigin
	sans       []string
	sansOrigin labelAssignmentOrigin
}

func putIndexedValue(values map[labelPathKey]indexedValue, key labelPathKey, value any, origin labelAssignmentOrigin) {
	if existing, exists := values[key]; exists && existing.origin > origin {
		return
	}
	values[key] = indexedValue{value: value, origin: origin}
}

func pathKeyForTypes(path []lexer.TokenType) labelPathKey {
	var builder strings.Builder
	for _, tokenType := range path {
		builder.WriteByte('/')
		builder.WriteString(strconv.Itoa(int(tokenType)))
	}
	return labelPathKey(builder.String())
}

func pathKeyForSegments(segments []ast.Segment) labelPathKey {
	var builder strings.Builder
	for _, segment := range segments {
		builder.WriteByte('/')
		builder.WriteString(strconv.Itoa(int(segment.Type)))
	}
	return labelPathKey(builder.String())
}

func (s *Resource) indexHeader(path []ast.Segment, value any, origin labelAssignmentOrigin) {
	if len(path) < 2 {
		return
	}

	name := path[len(path)-1]
	if name.Type != lexer.TokenIdentifier || name.Lexeme == "" {
		return
	}
	headerValue, ok := value.(string)
	if !ok {
		return
	}

	key := pathKeyForSegments(path[:len(path)-1])
	if s.headers[key] == nil {
		s.headers[key] = make(map[string]indexedString)
	}
	if existing, exists := s.headers[key][name.Lexeme]; exists && existing.origin > origin {
		return
	}
	s.headers[key][name.Lexeme] = indexedString{value: headerValue, origin: origin}
}

func (s *Resource) indexTLSDomain(path []ast.Segment, value any, origin labelAssignmentOrigin) {
	prefix, index, field, ok := indexedTLSDomainPath(path)
	if !ok {
		return
	}

	key := pathKeyForTypes(prefix)
	if s.tlsDomains[key] == nil {
		s.tlsDomains[key] = make(map[int]*indexedTLSDomain)
	}
	domain := s.tlsDomains[key][index]
	if domain == nil {
		domain = &indexedTLSDomain{}
		s.tlsDomains[key][index] = domain
	}

	switch field {
	case lexer.TokenMain:
		main, ok := value.(string)
		if !ok || domain.mainOrigin > origin {
			return
		}
		domain.main = main
		domain.mainOrigin = origin
	case lexer.TokenSANs:
		sans, ok := value.([]string)
		if !ok || domain.sansOrigin > origin {
			return
		}
		domain.sans = sans
		domain.sansOrigin = origin
	}
}

func indexedTLSDomainPath(path []ast.Segment) ([]lexer.TokenType, int, lexer.TokenType, bool) {
	for index, segment := range path {
		if segment.Type != lexer.TokenDomains {
			continue
		}
		domainIndex, ok := segment.Value.(int)
		if !ok || len(path) != index+2 {
			return nil, 0, lexer.TokenEOF, false
		}
		field := path[index+1].Type
		if field != lexer.TokenMain && field != lexer.TokenSANs {
			return nil, 0, lexer.TokenEOF, false
		}

		prefix := make([]lexer.TokenType, 0, index)
		for _, segment := range path[:index] {
			prefix = append(prefix, segment.Type)
		}
		return prefix, domainIndex, field, true
	}
	return nil, 0, lexer.TokenEOF, false
}
