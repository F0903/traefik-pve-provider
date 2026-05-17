package labels

import (
	"sort"
	"strings"
)

type TLSDomain struct {
	Main string
	SANs []string
}

func (s *Set) StringValue(key string) (string, bool) {
	value, ok := s.value(key)
	typed, isString := value.(string)
	if !ok || !isString || typed == "" {
		return "", false
	}
	return typed, true
}

func (s *Set) BoolValue(key string) (bool, bool) {
	value, ok := s.value(key)
	typed, isBool := value.(bool)
	return typed, ok && isBool
}

func (s *Set) value(key string) (any, bool) {
	value, ok := s.values[labelKey(key)]
	if !ok {
		return nil, false
	}
	return value.value, true
}

func (s *Resource) StringValue(key string) (string, bool) {
	value, ok := s.value(key)
	typed, isString := value.(string)
	if !ok || !isString || typed == "" {
		return "", false
	}
	return typed, true
}

func (s *Resource) BoolValue(key string) (bool, bool) {
	value, ok := s.value(key)
	typed, isBool := value.(bool)
	return typed, ok && isBool
}

func (s *Resource) IntValue(key string) (int, bool) {
	value, ok := s.value(key)
	typed, isInt := value.(int)
	return typed, ok && isInt
}

func (s *Resource) ListValue(key string) ([]string, bool) {
	value, ok := s.value(key)
	typed, isList := value.([]string)
	if !ok || !isList || len(typed) == 0 {
		return nil, false
	}
	return typed, true
}

func (s *Resource) value(key string) (any, bool) {
	if s == nil {
		return nil, false
	}

	value, ok := s.values[labelKey(key)]
	if !ok {
		return nil, false
	}
	return value.value, true
}

func (s *Resource) Headers(key string) map[string]string {
	if s == nil {
		return nil
	}

	values := s.headers[labelKey(key)]
	if len(values) == 0 {
		return nil
	}

	headers := make(map[string]string, len(values))
	for name, value := range values {
		headers[name] = value.value
	}
	return headers
}

func (s *Resource) TLSDomains(key string) []TLSDomain {
	if s == nil {
		return nil
	}

	domains := s.tlsDomains[labelKey(key)]
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

func labelKey(raw string) labelPathKey {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.Trim(raw, ".")
	return labelPathKey(raw)
}
