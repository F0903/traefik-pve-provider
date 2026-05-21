package traefik

import "strings"

const providerInterfacesKey = "pve.interfaces"

func providerInterfacePatterns(labels map[string]string) []string {
	raw, ok := labels[providerInterfacesKey]
	if !ok {
		return nil
	}

	parts := strings.Split(raw, ",")
	patterns := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		pattern := strings.TrimSpace(part)
		if pattern == "" || seen[pattern] {
			continue
		}
		seen[pattern] = true
		patterns = append(patterns, pattern)
	}
	return patterns
}
