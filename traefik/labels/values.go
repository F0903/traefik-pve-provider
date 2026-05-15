package labels

import (
	"sort"

	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
)

func (s *Set) StringValue(tokenType lexer.TokenType) (string, bool) {
	value, ok := s.value(tokenType)
	typed, isString := value.(string)
	if !ok || !isString || typed == "" {
		return "", false
	}
	return typed, true
}

func (s *Set) BoolValue(tokenType lexer.TokenType) (bool, bool) {
	value, ok := s.value(tokenType)
	typed, isBool := value.(bool)
	return typed, ok && isBool
}

func (s *Set) value(tokenType lexer.TokenType) (any, bool) {
	var selected labelAssignment
	found := false
	for _, assignment := range s.assignments {
		segments := assignment.assignment.Target.Segments()
		if len(segments) != 1 || segments[0].Type != tokenType {
			continue
		}
		if found && selected.origin > assignment.origin {
			continue
		}
		selected = assignment
		found = true
	}
	if !found {
		return nil, false
	}
	return assignmentValue(selected.assignment.Value)
}

func (s *Resource) StringValue(path ...lexer.TokenType) (string, bool) {
	value, ok := s.value(path...)
	typed, isString := value.(string)
	if !ok || !isString || typed == "" {
		return "", false
	}
	return typed, true
}

func (s *Resource) BoolValue(path ...lexer.TokenType) (bool, bool) {
	value, ok := s.value(path...)
	typed, isBool := value.(bool)
	return typed, ok && isBool
}

func (s *Resource) IntValue(path ...lexer.TokenType) (int, bool) {
	value, ok := s.value(path...)
	typed, isInt := value.(int)
	return typed, ok && isInt
}

func (s *Resource) ListValue(path ...lexer.TokenType) ([]string, bool) {
	value, ok := s.value(path...)
	typed, isList := value.([]string)
	if !ok || !isList || len(typed) == 0 {
		return nil, false
	}
	return typed, true
}

func (s *Resource) value(path ...lexer.TokenType) (any, bool) {
	if s == nil || s.labels == nil {
		return nil, false
	}

	var selected labelAssignment
	found := false
	for _, assignment := range s.labels.assignments {
		rest, ok := assignment.objectPath(s)
		if !ok || !segmentTypesEqual(rest, path) {
			continue
		}
		if found && selected.origin > assignment.origin {
			continue
		}
		selected = assignment
		found = true
	}
	if !found {
		return nil, false
	}
	return assignmentValue(selected.assignment.Value)
}

func (s *Resource) Headers(path ...lexer.TokenType) map[string]string {
	values := make(map[string]labelNamedValue)
	for _, assignment := range s.assignmentsWithPrefix(path...) {
		rest, ok := assignment.objectPath(s)
		if !ok || len(rest) != len(path)+1 {
			continue
		}
		name := rest[len(path)]
		if name.Type != lexer.TokenIdentifier || name.Lexeme == "" {
			continue
		}
		value, ok := assignmentValue(assignment.assignment.Value)
		if !ok {
			continue
		}
		headerValue, ok := value.(string)
		if !ok {
			continue
		}
		if existing, exists := values[name.Lexeme]; exists && existing.origin > assignment.origin {
			continue
		}
		values[name.Lexeme] = labelNamedValue{value: headerValue, origin: assignment.origin}
	}
	if len(values) == 0 {
		return nil
	}

	headers := make(map[string]string, len(values))
	for name, value := range values {
		headers[name] = value.value
	}
	return headers
}

func (s *Resource) TLSDomains(path ...lexer.TokenType) []TLSDomain {
	domains := make(map[int]*TLSDomain)
	origins := make(map[int]map[lexer.TokenType]labelAssignmentOrigin)
	domainPath := append(append([]lexer.TokenType{}, path...), lexer.TokenDomains)

	for _, assignment := range s.assignmentsWithPrefix(domainPath...) {
		rest, ok := assignment.objectPath(s)
		if !ok || len(rest) != len(path)+2 || rest[len(path)].Type != lexer.TokenDomains {
			continue
		}
		index, ok := rest[len(path)].Value.(int)
		if !ok {
			continue
		}
		field := rest[len(path)+1].Type
		if field != lexer.TokenMain && field != lexer.TokenSANs {
			continue
		}
		if origins[index] == nil {
			origins[index] = make(map[lexer.TokenType]labelAssignmentOrigin)
		}
		if existing, exists := origins[index][field]; exists && existing > assignment.origin {
			continue
		}
		value, ok := assignmentValue(assignment.assignment.Value)
		if !ok {
			continue
		}

		domain := domains[index]
		if domain == nil {
			domain = &TLSDomain{}
			domains[index] = domain
		}
		switch field {
		case lexer.TokenMain:
			main, ok := value.(string)
			if ok {
				domain.Main = main
				origins[index][field] = assignment.origin
			}
		case lexer.TokenSANs:
			sans, ok := value.([]string)
			if ok {
				domain.SANs = sans
				origins[index][field] = assignment.origin
			}
		}
	}

	if len(domains) == 0 {
		return nil
	}

	indices := make([]int, 0, len(domains))
	for index := range domains {
		indices = append(indices, index)
	}
	sort.Ints(indices)

	result := make([]TLSDomain, 0, len(indices))
	for _, index := range indices {
		result = append(result, *domains[index])
	}
	return result
}
