package inventory

import "github.com/F0903/traefik-pve-provider/metadata"

type Kind string

const (
	KindVM        Kind = "vm"
	KindContainer Kind = "container"
)

type Snapshot struct {
	Workloads []Workload
	Problems  []Problem
}

type Workload struct {
	Kind             Kind
	Node             string
	ID               int
	Name             string
	Status           string
	Tags             []string
	Notes            string
	TraefikLabels    map[string]string
	LabelDiagnostics []metadata.Diagnostic
	IPs              []IP
	Problems         []Problem
}

type IP struct {
	Address   string
	Version   int
	Prefix    int
	Interface string
}

type Problem struct {
	Node    string
	Kind    Kind
	ID      int
	Stage   string
	Message string
}

func (s Snapshot) TraefikEnabled() []Workload {
	enabled := make([]Workload, 0)
	for _, workload := range s.Workloads {
		if workload.TraefikLabels["traefik.enable"] == "true" {
			enabled = append(enabled, workload)
		}
	}
	return enabled
}
