package traefik

import (
	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
)

type PrepareOptions struct {
	ExtractMode labelcfg.ExtractMode
}

type PreparedSnapshot struct {
	Workloads []PreparedWorkload
	Problems  []inventory.Problem
}

type PreparedWorkload struct {
	inventory.Workload
	Labels LabelState
}

type LabelState struct {
	Raw                map[string]string
	Parsed             *labelcfg.Set
	ExtractDiagnostics []labelcfg.ExtractDiagnostic
	ParseDiagnostics   []labelcfg.Diagnostic
}

func Prepare(snapshot inventory.Snapshot, options PrepareOptions) PreparedSnapshot {
	extractor := labelcfg.Extractor{Prefix: labelcfg.DefaultPrefix, Mode: options.ExtractMode}
	prepared := PreparedSnapshot{
		Workloads: make([]PreparedWorkload, 0, len(snapshot.Workloads)),
		Problems:  snapshot.Problems,
	}

	for _, workload := range snapshot.Workloads {
		prepared.Workloads = append(prepared.Workloads, prepareWorkload(workload, extractor))
	}

	return prepared
}

func prepareWorkload(workload inventory.Workload, extractor labelcfg.Extractor) PreparedWorkload {
	extracted := extractor.Extract(workload.Notes)
	parsed, diagnostics := labelcfg.Parse(extracted.Labels, workloadObjectName(workload))
	return PreparedWorkload{
		Workload: workload,
		Labels: LabelState{
			Raw:                extracted.Labels,
			Parsed:             parsed,
			ExtractDiagnostics: extracted.Diagnostics,
			ParseDiagnostics:   diagnostics,
		},
	}
}

func (s *PreparedSnapshot) EnabledRunningWorkloads() []*inventory.Workload {
	workloads := make([]*inventory.Workload, 0)
	for index := range s.Workloads {
		workload := &s.Workloads[index]
		if workload.Status == "running" && workload.Labels.Enabled() {
			workloads = append(workloads, &workload.Workload)
		}
	}
	return workloads
}

func (s LabelState) Enabled() bool {
	return s.Parsed != nil && s.Parsed.Enabled()
}
