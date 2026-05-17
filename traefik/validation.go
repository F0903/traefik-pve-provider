package traefik

import (
	"fmt"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
)

func (b *configBuilder) validateLabels(workload PreparedWorkload) *labelcfg.Set {
	if workload.Labels.Parsed != nil {
		b.reportLabelDiagnostics(workload.Workload, workload.Labels.ParseDiagnostics)
		return workload.Labels.Parsed
	}

	set, diagnostics := labelcfg.Parse(workload.Labels.Raw, workloadObjectName(workload.Workload))
	b.reportLabelDiagnostics(workload.Workload, diagnostics)
	return set
}

func (b *configBuilder) reportLabelDiagnostics(workload inventory.Workload, diagnostics []labelcfg.Diagnostic) {
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
}
