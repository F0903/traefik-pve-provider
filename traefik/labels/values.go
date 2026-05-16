package labels

import (
	"sort"

	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
)

type TLSDomain struct {
	Main string
	SANs []string
}

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
	value, ok := s.values[pathKeyForTypes([]lexer.TokenType{tokenType})]
	if !ok {
		return nil, false
	}
	return value.value, true
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
	if s == nil {
		return nil, false
	}

	value, ok := s.values[pathKeyForTypes(path)]
	if !ok {
		return nil, false
	}
	return value.value, true
}

func (s *Resource) Headers(path ...lexer.TokenType) map[string]string {
	if s == nil {
		return nil
	}

	values := s.headers[pathKeyForTypes(path)]
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
	if s == nil {
		return nil
	}

	domains := s.tlsDomains[pathKeyForTypes(path)]
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
		domain := domains[index]
		result = append(result, TLSDomain{
			Main: domain.main,
			SANs: domain.sans,
		})
	}
	return result
}
