package traefik

import (
	"fmt"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
)

func (b *configBuilder) validateLabels(workload PreparedWorkload, names GeneratedNames) *labelcfg.Set {
	if workload.Labels.Parsed != nil {
		b.reportLabelDiagnostics(workload.Workload, workload.Labels.ParseDiagnostics)
		return workload.Labels.Parsed
	}

	labels, nameDiagnostics := labelsForGeneratedNames(workload.Labels.Raw, names.Base)
	b.reportNameDiagnostics(workload.Workload, nameDiagnostics)
	set, diagnostics := labelcfg.ParseWithSources(labels, names.Base, workload.Labels.Sources)
	b.reportLabelDiagnostics(workload.Workload, diagnostics)
	return set
}

func (b *configBuilder) reportNameDiagnostics(workload inventory.Workload, diagnostics []string) {
	for _, diagnostic := range diagnostics {
		b.addDiagnostic(workload, diagnostic)
	}
}

func (b *configBuilder) reportLabelDiagnostics(workload inventory.Workload, diagnostics []labelcfg.Diagnostic) {
	for _, diagnostic := range diagnostics {
		source := diagnostic.Source
		switch diagnostic.Err.Kind {
		case labelcfg.ErrInvalidBoolean:
			b.addDiagnosticWithSource(workload, fmt.Sprintf("label %q has invalid boolean value %q", diagnostic.Key, diagnostic.Value), source.Line, source.Fragment)
		case labelcfg.ErrInvalidInteger:
			b.addDiagnosticWithSource(workload, fmt.Sprintf("label %q has invalid integer value %q", diagnostic.Key, diagnostic.Value), source.Line, source.Fragment)
		default:
			b.addDiagnosticWithSource(workload, fmt.Sprintf("unsupported label %q ignored", diagnostic.Key), source.Line, source.Fragment)
		}
	}
}
