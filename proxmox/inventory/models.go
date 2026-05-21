package inventory

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
	Kind              Kind
	Node              string
	ID                int
	Name              string
	Status            string
	Tags              []string
	Notes             string
	IPs               []IP
	InterfacePatterns []string
	Problems          []Problem
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
