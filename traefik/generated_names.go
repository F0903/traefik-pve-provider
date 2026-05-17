package traefik

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
)

const maxGeneratedNameLength = 63
const labelNameKey = "traefik.name"

type GeneratedNames struct {
	Source      string
	Base        string
	Diagnostics []string
}

func generatedNamesForWorkload(workload inventory.Workload) GeneratedNames {
	raw := strings.TrimSpace(workload.Name)
	fallback := workloadFallbackName(workload)
	if raw == "" {
		return GeneratedNames{Source: fallback, Base: fallback}
	}

	normalized := normalizeGeneratedName(raw)
	if normalized == "" {
		return GeneratedNames{
			Source: raw,
			Base:   fallback,
			Diagnostics: []string{
				fmt.Sprintf("workload name %q cannot be used as a generated Traefik name; using %q", raw, fallback),
			},
		}
	}

	names := GeneratedNames{Source: raw, Base: normalized}
	if raw != normalized {
		names.Diagnostics = append(names.Diagnostics, fmt.Sprintf("workload name %q normalized to %q for generated Traefik names", raw, normalized))
	}
	return names
}

func generatedNamesForPrepared(workload PreparedWorkload) GeneratedNames {
	if workload.Names.Base != "" {
		return workload.Names
	}
	return generatedNamesForWorkload(workload.Workload)
}

func labelsForGeneratedNames(raw map[string]string, fallbackName string) (map[string]string, []string) {
	nameOverride, ok := raw[labelNameKey]
	if !ok {
		return raw, nil
	}

	trimmed := strings.TrimSpace(nameOverride)
	normalized := normalizeGeneratedName(trimmed)
	if normalized == "" {
		labels := cloneLabels(raw)
		delete(labels, labelNameKey)
		return labels, []string{
			fmt.Sprintf("label %q value %q cannot be used as a generated Traefik name; using %q", labelNameKey, nameOverride, fallbackName),
		}
	}
	if normalized == trimmed {
		return raw, nil
	}

	labels := cloneLabels(raw)
	labels[labelNameKey] = normalized
	return labels, []string{
		fmt.Sprintf("label %q value %q normalized to %q for generated Traefik names", labelNameKey, nameOverride, normalized),
	}
}

func cloneLabels(labels map[string]string) map[string]string {
	cloned := make(map[string]string, len(labels))
	for key, value := range labels {
		cloned[key] = value
	}
	return cloned
}

func workloadFallbackName(workload inventory.Workload) string {
	if workload.Kind != "" && workload.ID != 0 {
		return generatedNameWithSuffix(string(workload.Kind), strconv.Itoa(workload.ID))
	}
	return "workload"
}

func normalizeGeneratedName(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var builder strings.Builder
	lastDash := false
	for _, r := range raw {
		if isGeneratedNameChar(r) {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if builder.Len() > 0 && !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return trimGeneratedName(builder.String())
}

func generatedNameWithSuffix(base string, suffixes ...string) string {
	base = normalizeGeneratedName(base)
	suffix := normalizeGeneratedName(strings.Join(suffixes, "-"))
	if suffix == "" {
		return base
	}
	if base == "" {
		return suffix
	}

	separatorAndSuffix := "-" + suffix
	if len(separatorAndSuffix) >= maxGeneratedNameLength {
		return trimGeneratedName(suffix)
	}
	if len(base)+len(separatorAndSuffix) > maxGeneratedNameLength {
		base = trimGeneratedName(base[:maxGeneratedNameLength-len(separatorAndSuffix)])
	}
	return base + separatorAndSuffix
}

func trimGeneratedName(value string) string {
	value = strings.Trim(value, "-")
	if len(value) > maxGeneratedNameLength {
		value = strings.TrimRight(value[:maxGeneratedNameLength], "-")
	}
	return value
}

func isGeneratedNameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}
