package traefik

import (
	"fmt"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
)

func (b *configBuilder) validateLabels(workload inventory.Workload) *labelcfg.Set {
	defaultName := workloadObjectName(workload)
	set, diagnostics := labelcfg.Parse(workload.TraefikLabels, defaultName)
	for _, diagnostic := range diagnostics {
		switch diagnostic.Err.Kind {
		case labelcfg.ErrInvalidBoolean:
			b.addDiagnostic(workload, fmt.Sprintf("label %q has invalid boolean value %q", diagnostic.Key, diagnostic.Value))
		case labelcfg.ErrInvalidInteger:
			b.addDiagnostic(workload, fmt.Sprintf("label %q has invalid integer value %q", diagnostic.Key, diagnostic.Value))
		default:
			b.addDiagnostic(workload, fmt.Sprintf("unsupported label %q ignored", diagnostic.Key))
		}
	}
	return set
}
