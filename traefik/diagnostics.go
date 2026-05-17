package traefik

import "github.com/F0903/traefik-pve-provider/proxmox/inventory"

type Diagnostic struct {
	Node    string
	Kind    inventory.Kind
	ID      int
	Message string
}

func (b *configBuilder) addDiagnostic(workload inventory.Workload, message string) {
	b.diagnostics = append(b.diagnostics, Diagnostic{
		Node:    workload.Node,
		Kind:    workload.Kind,
		ID:      workload.ID,
		Message: message,
	})
}
