package scanner

import "strings"

func (s *Scanner) includedNode(node string) bool {
	if len(s.includedNodes) == 0 {
		return true
	}
	return s.includedNodes[strings.ToLower(strings.TrimSpace(node))]
}

func (s *Scanner) matchesRequiredTags(tags []string) bool {
	if len(s.requiredTags) == 0 {
		return true
	}

	tagSet := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tagSet[strings.ToLower(tag)] = true
	}
	for _, required := range s.requiredTags {
		if !tagSet[required] {
			return false
		}
	}
	return true
}

func splitTags(tags string) []string {
	if tags == "" {
		return nil
	}

	raw := strings.FieldsFunc(tags, func(r rune) bool {
		return r == ';' || r == ',' || r == ' '
	})
	result := make([]string, 0, len(raw))
	for _, tag := range raw {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			result = append(result, tag)
		}
	}
	return result
}

func normalizedNodeNames(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}

func normalizedSet(values []string) map[string]bool {
	list := normalizedList(values)
	if len(list) == 0 {
		return nil
	}
	set := make(map[string]bool, len(list))
	for _, value := range list {
		set[value] = true
	}
	return set
}

func normalizedList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func normalizedMaxConcurrency(value int) int {
	if value <= 0 {
		return defaultMaxConcurrency
	}
	return value
}
