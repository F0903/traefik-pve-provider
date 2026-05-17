package traefik

import "github.com/F0903/traefik-pve-provider/proxmox/inventory"

type Diagnostic struct {
	Node     string
	Kind     inventory.Kind
	ID       int
	Message  string
	Line     int
	Fragment string
}

func (b *configBuilder) addDiagnostic(workload inventory.Workload, message string) {
	b.addDiagnosticWithSource(workload, message, 0, "")
}

func (b *configBuilder) addDiagnosticWithSource(workload inventory.Workload, message string, line int, fragment string) {
	b.diagnostics = append(b.diagnostics, Diagnostic{
		Node:     workload.Node,
		Kind:     workload.Kind,
		ID:       workload.ID,
		Message:  message,
		Line:     line,
		Fragment: fragment,
	})
}
