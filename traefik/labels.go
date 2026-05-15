package traefik

import (
	"fmt"
	"net"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
)

func namesFromLabels(labels map[string]string, prefix string, supportedOption func(string) bool) []string {
	names, _ := explicitNamesFromLabels(labels, prefix, supportedOption)
	return names
}

func explicitNamesFromLabels(labels map[string]string, prefix string, supportedOption func(string) bool) ([]string, bool) {
	seen := make(map[string]bool)
	found := false
	for key := range labels {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		rest := strings.TrimPrefix(key, prefix)
		name, option, ok := strings.Cut(rest, ".")
		if ok && name != "" {
			found = true
		}
		if ok && name != "" && supportedOption(option) {
			seen[name] = true
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, found
}

func defaultObjectName(workload inventory.Workload) string {
	if name := strings.TrimSpace(workload.TraefikLabels["traefik.name"]); name != "" {
		return name
	}
	if name := strings.TrimSpace(workload.Name); name != "" {
		return name
	}
	if workload.Kind != "" && workload.ID != 0 {
		return fmt.Sprintf("%s-%d", workload.Kind, workload.ID)
	}
	return "workload"
}

func defaultHTTPRule(name string, options Options) string {
	host := name
	if domain := strings.Trim(strings.TrimSpace(options.DefaultDomain), "."); domain != "" {
		host += "." + domain
	}
	return fmt.Sprintf("Host(`%s`)", host)
}

func serverURL(scheme, host, port string) string {
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("%s://%s:%s", scheme, host, port)
}

func serverAddress(host, port string) string {
	return net.JoinHostPort(host, port)
}

func isEnabled(labels map[string]string) bool {
	enabled, ok := parseBool(labels["traefik.enable"])
	return ok && enabled
}

func hasLabelPrefix(labels map[string]string, prefix string) bool {
	for key := range labels {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func labelOrDefault(labels map[string]string, key, fallback string) string {
	if value := strings.TrimSpace(labels[key]); value != "" {
		return value
	}
	return fallback
}

func firstLabelOrDefault(labels map[string]string, fallback string, keys ...string) string {
	if value := firstLabel(labels, keys...); value != "" {
		return value
	}
	return fallback
}

func firstLabel(labels map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(labels[key]); value != "" {
			return value
		}
	}
	return ""
}

func labelsWithPrefix(labels map[string]string, prefix string) map[string]string {
	values := make(map[string]string)
	for key, value := range labels {
		name, found := strings.CutPrefix(key, prefix)
		if !found || name == "" {
			continue
		}
		if value = strings.TrimSpace(value); value != "" {
			values[name] = value
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func containsString(values []string, target string) bool {
	return slices.Contains(values, target)
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func parseBool(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "on":
		return true, true
	case "false", "0", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func parseInt(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.Atoi(raw)
	return value, err == nil
}
