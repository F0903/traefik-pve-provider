package scanner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/F0903/traefik-pve-provider/proxmox"
	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
)

var fakeAPICallsMu sync.Mutex

type fakeAPI struct {
	calls                 map[string]int
	configHook            func(int)
	nodesErr              error
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
	f.recordCall("nodes")
	if f.nodesErr != nil {
		return nil, f.nodesErr
	}
	return f.nodes, nil
}

func (f fakeAPI) VirtualMachines(ctx context.Context, node string) ([]proxmox.Resource, error) {
	f.recordCall("vms:" + node)
	return f.vms[node], nil
}

func (f fakeAPI) Containers(ctx context.Context, node string) ([]proxmox.Resource, error) {
	f.recordCall("containers:" + node)
	return f.containers[node], nil
}

func (f fakeAPI) VMConfig(ctx context.Context, node string, vmid int) (proxmox.GuestConfig, error) {
	f.recordCall(fmt.Sprintf("vm-config:%d", vmid))
	if f.configHook != nil {
		f.configHook(vmid)
	}
	if err := f.configErr[vmid]; err != nil {
		return proxmox.GuestConfig{}, err
	}
	return f.vmConfigs[vmid], nil
}

func (f fakeAPI) ContainerConfig(ctx context.Context, node string, vmid int) (proxmox.GuestConfig, error) {
	f.recordCall(fmt.Sprintf("container-config:%d", vmid))
	if f.configHook != nil {
		f.configHook(vmid)
	}
	if err := f.configErr[vmid]; err != nil {
		return proxmox.GuestConfig{}, err
	}
	return f.containerConfigs[vmid], nil
}

func (f fakeAPI) VMNetworkInterfaces(ctx context.Context, node string, vmid int) (proxmox.GuestAgentInterfaces, error) {
	f.recordCall(fmt.Sprintf("vm-interfaces:%d", vmid))
	if err := f.vmInterfaceErr[vmid]; err != nil {
		return proxmox.GuestAgentInterfaces{}, err
	}
	return f.vmInterfaces[vmid], nil
}

func (f fakeAPI) ContainerInterfaces(ctx context.Context, node string, vmid int) ([]proxmox.NetworkInterface, error) {
	f.recordCall(fmt.Sprintf("container-interfaces:%d", vmid))
	if err := f.containerInterfaceErr[vmid]; err != nil {
		return nil, err
	}
	return f.containerInterfaces[vmid], nil
}

func (f fakeAPI) recordCall(key string) {
	if f.calls != nil {
		fakeAPICallsMu.Lock()
		defer fakeAPICallsMu.Unlock()
		f.calls[key]++
	}
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

	scanner := New(api, Options{})
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(snapshot.Workloads) != 2 {
		t.Fatalf("workload count = %d, want 2", len(snapshot.Workloads))
	}

	vm := snapshot.Workloads[0]
	if vm.Kind != inventory.KindVM || vm.Node != "pve-1" || vm.ID != 100 || vm.Name != "app-vm" {
		t.Fatalf("vm workload = %#v", vm)
	}
	if !strings.Contains(vm.Notes, "http.routers.app.rule=Host(`app.example.com`)") {
		t.Fatalf("notes = %q", vm.Notes)
	}
	if len(vm.IPs) != 0 {
		t.Fatalf("vm IPs after Scan() = %#v, want unresolved", vm.IPs)
	}

	scanner.ResolveIPs(context.Background(), []*inventory.Workload{&snapshot.Workloads[0]})
	vm = snapshot.Workloads[0]
	if len(vm.IPs) != 1 || vm.IPs[0].Address != "192.168.10.20" {
		t.Fatalf("vm IPs after ResolveIPs() = %#v", vm.IPs)
	}

	if len(vm.Tags) != 2 || vm.Tags[0] != "override" || vm.Tags[1] != "tag" {
		t.Fatalf("vm tags = %#v", vm.Tags)
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

	scanner := New(api, Options{SkipIPResolution: true})
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

	scanner := New(api, Options{SkipStopped: true, SkipIPResolution: true})
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

	scanner := New(api, Options{SkipIPResolution: true})
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

	scanner := New(api, Options{
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

func TestScannerSkipsNodeListingWhenNodesConfigured(t *testing.T) {
	calls := make(map[string]int)
	api := fakeAPI{
		calls:    calls,
		nodesErr: errors.New("nodes should not be listed"),
		vms: map[string][]proxmox.Resource{
			"pve-1": {{VMID: 100, Name: "web", Status: "running"}},
		},
		containers: map[string][]proxmox.Resource{},
		vmConfigs: map[int]proxmox.GuestConfig{
			100: {Description: "```traefik\nenable=true\n```"},
		},
		containerConfigs:    map[int]proxmox.GuestConfig{},
		vmInterfaces:        map[int]proxmox.GuestAgentInterfaces{},
		containerInterfaces: map[int][]proxmox.NetworkInterface{},
		configErr:           map[int]error{},
	}

	scanner := New(api, Options{
		SkipIPResolution: true,
		Nodes:            []string{"pve-1"},
	})
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if calls["nodes"] != 0 {
		t.Fatalf("nodes calls = %d, want 0", calls["nodes"])
	}
	if calls["vms:pve-1"] != 1 {
		t.Fatalf("vm list calls = %d, want 1", calls["vms:pve-1"])
	}
	if len(snapshot.Workloads) != 1 || snapshot.Workloads[0].ID != 100 {
		t.Fatalf("workloads = %#v", snapshot.Workloads)
	}
}

func TestScannerFiltersRequiredTagsBeforeGuestCalls(t *testing.T) {
	calls := make(map[string]int)
	api := fakeAPI{
		calls: calls,
		nodes: []proxmox.Node{{Name: "pve-1", Status: "online"}},
		vms: map[string][]proxmox.Resource{
			"pve-1": {
				{VMID: 100, Name: "db", Status: "running", Tags: "prod"},
				{VMID: 101, Name: "web", Status: "running", Tags: "traefik;prod"},
			},
		},
		containers: map[string][]proxmox.Resource{
			"pve-1": {
				{VMID: 200, Name: "cache", Status: "running", Tags: "prod"},
			},
		},
		vmConfigs: map[int]proxmox.GuestConfig{
			101: {Description: "```traefik\nenable=true\n```"},
		},
		containerConfigs: map[int]proxmox.GuestConfig{},
		vmInterfaces: map[int]proxmox.GuestAgentInterfaces{
			101: {
				Result: []proxmox.NetworkInterface{{Name: "eth0", IPAddresses: []proxmox.IPAddress{{Address: "10.0.0.10", Type: "ipv4"}}}},
			},
		},
		containerInterfaces: map[int][]proxmox.NetworkInterface{},
		configErr:           map[int]error{},
	}

	scanner := New(api, Options{RequiredTags: []string{"traefik"}})
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(snapshot.Workloads) != 1 || snapshot.Workloads[0].ID != 101 {
		t.Fatalf("workloads = %#v", snapshot.Workloads)
	}
	scanner.ResolveIPs(context.Background(), []*inventory.Workload{&snapshot.Workloads[0]})
	if calls["vm-config:100"] != 0 || calls["vm-interfaces:100"] != 0 {
		t.Fatalf("excluded vm calls: config=%d interfaces=%d", calls["vm-config:100"], calls["vm-interfaces:100"])
	}
	if calls["container-config:200"] != 0 || calls["container-interfaces:200"] != 0 {
		t.Fatalf("excluded container calls: config=%d interfaces=%d", calls["container-config:200"], calls["container-interfaces:200"])
	}
	if calls["vm-config:101"] != 1 || calls["vm-interfaces:101"] != 1 {
		t.Fatalf("included vm calls: config=%d interfaces=%d", calls["vm-config:101"], calls["vm-interfaces:101"])
	}
}

func TestScannerDoesNotResolveIPsDuringScan(t *testing.T) {
	calls := make(map[string]int)
	api := fakeAPI{
		calls: calls,
		nodes: []proxmox.Node{{Name: "pve-1", Status: "online"}},
		vms: map[string][]proxmox.Resource{
			"pve-1": {{VMID: 100, Name: "disabled", Status: "running"}},
		},
		containers: map[string][]proxmox.Resource{},
		vmConfigs: map[int]proxmox.GuestConfig{
			100: {Description: "```traefik\nenable=false\n```"},
		},
		containerConfigs: map[int]proxmox.GuestConfig{},
		vmInterfaceErr: map[int]error{
			100: errors.New("interfaces should not be fetched"),
		},
		containerInterfaces: map[int][]proxmox.NetworkInterface{},
		configErr:           map[int]error{},
	}

	scanner := New(api, Options{})
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(snapshot.Workloads) != 1 || snapshot.Workloads[0].ID != 100 {
		t.Fatalf("workloads = %#v", snapshot.Workloads)
	}
	if calls["vm-interfaces:100"] != 0 {
		t.Fatalf("interface calls = %d, want 0", calls["vm-interfaces:100"])
	}
	if len(snapshot.Workloads[0].Problems) != 0 {
		t.Fatalf("problems = %#v", snapshot.Workloads[0].Problems)
	}
}

func TestScannerLimitsConcurrentGuestConfigCalls(t *testing.T) {
	var mu sync.Mutex
	active := 0
	maxActive := 0
	configHook := func(int) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()

		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		active--
		mu.Unlock()
	}

	api := fakeAPI{
		configHook: configHook,
		nodes:      []proxmox.Node{{Name: "pve-1", Status: "online"}},
		vms: map[string][]proxmox.Resource{
			"pve-1": {
				{VMID: 100, Name: "vm-1", Status: "running"},
				{VMID: 101, Name: "vm-2", Status: "running"},
				{VMID: 102, Name: "vm-3", Status: "running"},
				{VMID: 103, Name: "vm-4", Status: "running"},
			},
		},
		containers: map[string][]proxmox.Resource{},
		vmConfigs: map[int]proxmox.GuestConfig{
			100: {}, 101: {}, 102: {}, 103: {},
		},
		containerConfigs:    map[int]proxmox.GuestConfig{},
		vmInterfaces:        map[int]proxmox.GuestAgentInterfaces{},
		containerInterfaces: map[int][]proxmox.NetworkInterface{},
		configErr:           map[int]error{},
	}

	scanner := New(api, Options{SkipIPResolution: true, MaxConcurrency: 2})
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(snapshot.Workloads) != 4 {
		t.Fatalf("workloads = %#v", snapshot.Workloads)
	}
	if maxActive > 2 {
		t.Fatalf("max active config calls = %d, want at most 2", maxActive)
	}
	if maxActive < 2 {
		t.Fatalf("max active config calls = %d, want concurrency to be used", maxActive)
	}
	for index, workload := range snapshot.Workloads {
		wantID := 100 + index
		if workload.ID != wantID {
			t.Fatalf("workload order at %d = %d, want %d", index, workload.ID, wantID)
		}
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

	scanner := New(api, Options{SkipIPResolution: true})
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
		containers: map[string][]proxmox.Resource{},
		vmConfigs: map[int]proxmox.GuestConfig{
			100: {Description: "```traefik\nenable=true\n```"},
		},
		containerConfigs: map[int]proxmox.GuestConfig{},
		vmInterfaceErr: map[int]error{
			100: &proxmox.APIError{Method: "GET", Path: "/agent/network-get-interfaces", StatusCode: 403},
		},
		containerInterfaces: map[int][]proxmox.NetworkInterface{},
		configErr:           map[int]error{},
	}

	scanner := New(api, Options{})
	snapshot, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	scanner.ResolveIPs(context.Background(), []*inventory.Workload{&snapshot.Workloads[0]})

	if len(snapshot.Workloads) != 1 || len(snapshot.Workloads[0].Problems) != 1 {
		t.Fatalf("workloads = %#v", snapshot.Workloads)
	}
	if !strings.Contains(snapshot.Workloads[0].Problems[0].Message, "VM.GuestAgent.Audit") {
		t.Fatalf("problem = %#v", snapshot.Workloads[0].Problems[0])
	}
}
