package traefik

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
)

func defaultObjectName(workload inventory.Workload, labels *labelcfg.Set) string {
	if name, ok := labels.NameOverride(); ok {
		return name
	}
	return workloadObjectName(workload)
}

func workloadObjectName(workload inventory.Workload) string {
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

func (b *configBuilder) claimObjectNames(workload inventory.Workload, names []string, found bool, claimed map[string]objectOwner, objectKind string) ([]string, bool) {
	if len(names) == 0 {
		return nil, found
	}

	usable := make([]string, 0, len(names))
	for _, name := range names {
		if b.claimObjectName(name, workload, claimed, objectKind) {
			usable = append(usable, name)
		}
	}
	return usable, true
}

func (b *configBuilder) claimDefaultObjectName(name string, workload inventory.Workload, claimed map[string]objectOwner, objectKind string) string {
	if b.claimObjectName(name, workload, claimed, objectKind) {
		return name
	}

	for _, candidate := range defaultNameCollisionCandidates(name, workload) {
		if b.claimObjectName(candidate, workload, claimed, objectKind) {
			b.addDiagnostic(workload, fmt.Sprintf("%s name %q already exists; using %q", objectKind, name, candidate))
			return candidate
		}
	}

	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s-%d-%d", name, workload.ID, index)
		if b.claimObjectName(candidate, workload, claimed, objectKind) {
			b.addDiagnostic(workload, fmt.Sprintf("%s name %q already exists; using %q", objectKind, name, candidate))
			return candidate
		}
	}
}

func (b *configBuilder) claimObjectName(name string, workload inventory.Workload, claimed map[string]objectOwner, objectKind string) bool {
	owner := ownerFor(workload)
	if existing, exists := claimed[name]; exists {
		if existing == owner {
			return true
		}
		b.addDiagnostic(workload, fmt.Sprintf("%s name %q already used by %s; skipping duplicate", objectKind, name, existing.describe()))
		return false
	}
	claimed[name] = owner
	return true
}

func defaultNameCollisionCandidates(name string, workload inventory.Workload) []string {
	candidates := make([]string, 0, 3)
	if workload.ID != 0 {
		candidates = append(candidates, fmt.Sprintf("%s-%d", name, workload.ID))
	}
	if workload.Kind != "" && workload.ID != 0 {
		candidates = append(candidates, fmt.Sprintf("%s-%s-%d", name, workload.Kind, workload.ID))
	}
	if workload.Node != "" && workload.ID != 0 {
		candidates = append(candidates, fmt.Sprintf("%s-%s-%d", name, workload.Node, workload.ID))
	}
	return candidates
}

func ownerFor(workload inventory.Workload) objectOwner {
	return objectOwner{
		Node: workload.Node,
		Kind: workload.Kind,
		ID:   workload.ID,
		Name: workload.Name,
	}
}

func (o objectOwner) describe() string {
	parts := make([]string, 0, 4)
	if o.Node != "" {
		parts = append(parts, "node="+o.Node)
	}
	if o.Kind != "" {
		parts = append(parts, "kind="+string(o.Kind))
	}
	if o.ID != 0 {
		parts = append(parts, "id="+strconv.Itoa(o.ID))
	}
	if o.Name != "" {
		parts = append(parts, "name="+o.Name)
	}
	return strings.Join(parts, " ")
}
