package inventory

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/F0903/traefik-pve-provider/proxmox"
)

type fakeAPI struct {
	nodes                 []proxmox.Node
	vms                   map[string][]proxmox.Resource
	containers            map[string][]proxmox.Resource
	vmConfigs             map[int]proxmox.GuestConfig
	containerConfigs      map[int]proxmox.GuestConfig
	vmInterfaces          map[int]proxmox.GuestAgentInterfaces
	vmInterfaceErr        map[int]error
	containerInterfaces   map[int][]proxmox.NetworkInterface
	containerInterfaceErr map[int]error
	configErr             map[int]error
}

func (f fakeAPI) Nodes(ctx context.Context) ([]proxmox.Node, error) {
	return f.nodes, nil
}

func (f fakeAPI) VirtualMachines(ctx context.Context, node string) ([]proxmox.Resource, error) {
	return f.vms[node], nil
}

func (f fakeAPI) Containers(ctx context.Context, node string) ([]proxmox.Resource, error) {
	return f.containers[node], nil
}

func (f fakeAPI) VMConfig(ctx context.Context, node string, vmid int) (proxmox.GuestConfig, error) {
	if err := f.configErr[vmid]; err != nil {
		return proxmox.GuestConfig{}, err
	}
	return f.vmConfigs[vmid], nil
}

func (f fakeAPI) ContainerConfig(ctx context.Context, node string, vmid int) (proxmox.GuestConfig, error) {
	if err := f.configErr[vmid]; err != nil {
		return proxmox.GuestConfig{}, err
	}
	return f.containerConfigs[vmid], nil
}

func (f fakeAPI) VMNetworkInterfaces(ctx context.Context, node string, vmid int) (proxmox.GuestAgentInterfaces, error) {
	if err := f.vmInterfaceErr[vmid]; err != nil {
		return proxmox.GuestAgentInterfaces{}, err
	}
	return f.vmInterfaces[vmid], nil
}

func (f fakeAPI) ContainerInterfaces(ctx context.Context, node string, vmid int) ([]proxmox.NetworkInterface, error) {
	if err := f.containerInterfaceErr[vmid]; err != nil {
		return nil, err
	}
	return f.containerInterfaces[vmid], nil
}

func TestScannerBuildsWorkloadSnapshot(t *testing.T) {
	api := fakeAPI{
		nodes: []proxmox.Node{{Name: "pve-1"}},
		vms: map[string][]proxmox.Resource{
			"pve-1": {{VMID: 100, Name: "app-vm", Status: "running", Tags: "prod;web"}},
		},
		containers: map[string][]proxmox.Resource{
			"pve-1": {{VMID: 200, Name: "app-ct", Status: "stopped"}},
		},
		vmConfigs: map[int]proxmox.GuestConfig{
			100: {
				Description: "```traefik\nenable=true\nhttp.routers.app.rule=Host(`app.example.com`)\n```",
				Tags:        "override;tag",
			},
		},
		containerConfigs: map[int]proxmox.GuestConfig{
			200: {
				Description: "```traefik\nenable=false\n```",
			},
		},
		vmInterfaces: map[int]proxmox.GuestAgentInterfaces{
			100: {
				Result: []proxmox.NetworkInterface{
					{
						Name: "eth0",
						IPAddresses: []proxmox.IPAddress{
							{Address: "127.0.0.1", Type: "ipv4"},
							{Address: "192.168.10.20", Type: "ipv4"},
						},
					},
				},
			},
		},
		containerInterfaces: map[int][]proxmox.NetworkInterface{
			200: {{Name: "eth0", Inet: "10.0.0.4/24"}},
		},
		configErr: map[int]error{},
	}

	scanner := NewScanner(api, ScanOptions{})
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(snapshot.Workloads) != 2 {
		t.Fatalf("workload count = %d, want 2", len(snapshot.Workloads))
	}

	vm := snapshot.Workloads[0]
	if vm.Kind != KindVM || vm.Node != "pve-1" || vm.ID != 100 || vm.Name != "app-vm" {
		t.Fatalf("vm workload = %#v", vm)
	}
	if got := vm.TraefikLabels["traefik.http.routers.app.rule"]; got != "Host(`app.example.com`)" {
		t.Fatalf("rule = %q", got)
	}
	if len(vm.IPs) != 1 || vm.IPs[0].Address != "192.168.10.20" {
		t.Fatalf("vm IPs = %#v", vm.IPs)
	}
	if len(vm.Tags) != 2 || vm.Tags[0] != "override" || vm.Tags[1] != "tag" {
		t.Fatalf("vm tags = %#v", vm.Tags)
	}

	enabled := snapshot.TraefikEnabled()
	if len(enabled) != 1 || enabled[0].ID != 100 {
		t.Fatalf("enabled workloads = %#v", enabled)
	}
}

func TestScannerRecordsConfigProblem(t *testing.T) {
	api := fakeAPI{
		nodes: []proxmox.Node{{Name: "pve-1"}},
		vms: map[string][]proxmox.Resource{
			"pve-1": {{VMID: 100, Name: "broken-vm", Status: "running"}},
		},
		containers:       map[string][]proxmox.Resource{},
		vmConfigs:        map[int]proxmox.GuestConfig{},
		containerConfigs: map[int]proxmox.GuestConfig{},
		configErr: map[int]error{
			100: errors.New("permission denied"),
		},
	}

	scanner := NewScanner(api, ScanOptions{SkipIPResolution: true})
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(snapshot.Workloads) != 1 {
		t.Fatalf("workload count = %d", len(snapshot.Workloads))
	}
	if len(snapshot.Workloads[0].Problems) != 1 {
		t.Fatalf("problems = %#v", snapshot.Workloads[0].Problems)
	}
	if snapshot.Workloads[0].Problems[0].Stage != "config" {
		t.Fatalf("problem stage = %q", snapshot.Workloads[0].Problems[0].Stage)
	}
}

func TestScannerCanSkipStoppedWorkloads(t *testing.T) {
	api := fakeAPI{
		nodes: []proxmox.Node{{Name: "pve-1"}},
		vms: map[string][]proxmox.Resource{
			"pve-1": {
				{VMID: 100, Name: "running-vm", Status: "running"},
				{VMID: 101, Name: "stopped-vm", Status: "stopped"},
			},
		},
		containers:          map[string][]proxmox.Resource{},
		vmConfigs:           map[int]proxmox.GuestConfig{100: {}, 101: {}},
		containerConfigs:    map[int]proxmox.GuestConfig{},
		vmInterfaces:        map[int]proxmox.GuestAgentInterfaces{},
		containerInterfaces: map[int][]proxmox.NetworkInterface{},
		configErr:           map[int]error{},
	}

	scanner := NewScanner(api, ScanOptions{SkipStopped: true, SkipIPResolution: true})
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(snapshot.Workloads) != 1 || snapshot.Workloads[0].ID != 100 {
		t.Fatalf("workloads = %#v", snapshot.Workloads)
	}
}

func TestScannerSkipsOfflineNodes(t *testing.T) {
	api := fakeAPI{
		nodes: []proxmox.Node{
			{Name: "pve-online", Status: "online"},
			{Name: "pve-offline", Status: "offline"},
		},
		vms: map[string][]proxmox.Resource{
			"pve-online":  {{VMID: 100, Name: "running-vm", Status: "running"}},
			"pve-offline": {{VMID: 101, Name: "offline-vm", Status: "running"}},
		},
		containers:          map[string][]proxmox.Resource{},
		vmConfigs:           map[int]proxmox.GuestConfig{100: {}, 101: {}},
		containerConfigs:    map[int]proxmox.GuestConfig{},
		vmInterfaces:        map[int]proxmox.GuestAgentInterfaces{},
		containerInterfaces: map[int][]proxmox.NetworkInterface{},
		configErr:           map[int]error{},
	}

	scanner := NewScanner(api, ScanOptions{SkipIPResolution: true})
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(snapshot.Workloads) != 1 || snapshot.Workloads[0].ID != 100 {
		t.Fatalf("workloads = %#v", snapshot.Workloads)
	}
}

func TestScannerCanFilterNodesAndRequiredTags(t *testing.T) {
	api := fakeAPI{
		nodes: []proxmox.Node{
			{Name: "pve-1", Status: "online"},
			{Name: "pve-2", Status: "online"},
		},
		vms: map[string][]proxmox.Resource{
			"pve-1": {
				{VMID: 100, Name: "web", Status: "running", Tags: "traefik;prod"},
				{VMID: 101, Name: "db", Status: "running", Tags: "prod"},
			},
			"pve-2": {
				{VMID: 200, Name: "remote-web", Status: "running", Tags: "traefik;prod"},
			},
		},
		containers:          map[string][]proxmox.Resource{},
		vmConfigs:           map[int]proxmox.GuestConfig{100: {}, 101: {}, 200: {}},
		containerConfigs:    map[int]proxmox.GuestConfig{},
		vmInterfaces:        map[int]proxmox.GuestAgentInterfaces{},
		containerInterfaces: map[int][]proxmox.NetworkInterface{},
		configErr:           map[int]error{},
	}

	scanner := NewScanner(api, ScanOptions{
		SkipIPResolution: true,
		Nodes:            []string{"pve-1"},
		RequiredTags:     []string{"traefik"},
	})
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(snapshot.Workloads) != 1 || snapshot.Workloads[0].ID != 100 {
		t.Fatalf("workloads = %#v", snapshot.Workloads)
	}
}

func TestScannerUsesContainerHostnameFromConfig(t *testing.T) {
	api := fakeAPI{
		nodes: []proxmox.Node{{Name: "pve-1", Status: "online"}},
		vms:   map[string][]proxmox.Resource{},
		containers: map[string][]proxmox.Resource{
			"pve-1": {{VMID: 200, Status: "running"}},
		},
		vmConfigs: map[int]proxmox.GuestConfig{},
		containerConfigs: map[int]proxmox.GuestConfig{
			200: {Hostname: "oci-app"},
		},
		vmInterfaces:        map[int]proxmox.GuestAgentInterfaces{},
		containerInterfaces: map[int][]proxmox.NetworkInterface{},
		configErr:           map[int]error{},
	}

	scanner := NewScanner(api, ScanOptions{SkipIPResolution: true})
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(snapshot.Workloads) != 1 || snapshot.Workloads[0].Name != "oci-app" {
		t.Fatalf("workloads = %#v", snapshot.Workloads)
	}
}

func TestScannerAddsPermissionHintForForbiddenInterfaceLookup(t *testing.T) {
	api := fakeAPI{
		nodes: []proxmox.Node{{Name: "pve-1", Status: "online"}},
		vms: map[string][]proxmox.Resource{
			"pve-1": {{VMID: 100, Name: "vm", Status: "running"}},
		},
		containers:       map[string][]proxmox.Resource{},
		vmConfigs:        map[int]proxmox.GuestConfig{100: {}},
		containerConfigs: map[int]proxmox.GuestConfig{},
		vmInterfaceErr: map[int]error{
			100: &proxmox.APIError{Method: "GET", Path: "/agent/network-get-interfaces", StatusCode: 403},
		},
		containerInterfaces: map[int][]proxmox.NetworkInterface{},
		configErr:           map[int]error{},
	}

	scanner := NewScanner(api, ScanOptions{})
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(snapshot.Workloads) != 1 || len(snapshot.Workloads[0].Problems) != 1 {
		t.Fatalf("workloads = %#v", snapshot.Workloads)
	}
	if !strings.Contains(snapshot.Workloads[0].Problems[0].Message, "VM.GuestAgent.Audit") {
		t.Fatalf("problem = %#v", snapshot.Workloads[0].Problems[0])
	}
}
