package traefik

import (
	"testing"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
)

func TestPrepareAppliesProviderInterfacePatterns(t *testing.T) {
	prepared := Prepare(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				Kind:   inventory.KindContainer,
				Node:   "pve1",
				ID:     100,
				Name:   "app",
				Status: "running",
				Notes:  "```traefik\nenable=true\npve.interfaces=eth*, ens18, eth*\n```",
			},
		},
	}, PrepareOptions{})

	if len(prepared.Workloads) != 1 {
		t.Fatalf("workloads = %#v", prepared.Workloads)
	}
	patterns := prepared.Workloads[0].Workload.InterfacePatterns
	if len(patterns) != 2 || patterns[0] != "eth*" || patterns[1] != "ens18" {
		t.Fatalf("interface patterns = %#v", patterns)
	}
	if !prepared.Workloads[0].Labels.Enabled() {
		t.Fatal("workload labels were not enabled")
	}
	if len(prepared.Workloads[0].Labels.ParseDiagnostics) != 0 {
		t.Fatalf("parse diagnostics = %#v", prepared.Workloads[0].Labels.ParseDiagnostics)
	}
}
