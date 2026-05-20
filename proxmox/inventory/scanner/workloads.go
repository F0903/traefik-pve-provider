package scanner

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/F0903/traefik-pve-provider/proxmox"
	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
)

func (s *Scanner) scanClusterResources(ctx context.Context) (inventory.Snapshot, bool) {
	resources, err := s.api.ClusterResources(ctx)
	if err != nil {
		return inventory.Snapshot{}, false
	}
	if !clusterResourcesUsable(resources) {
		return inventory.Snapshot{}, false
	}

	resources = s.filteredClusterResources(resources)
	// Yaegi does not expose slices.SortFunc during interpreted plugin imports.
	sort.Slice(resources, func(i, j int) bool {
		return compareClusterResources(resources[i], resources[j]) < 0
	})

	snapshot := inventory.Snapshot{
		Workloads: s.scanWorkloads(len(resources), func(index int) inventory.Workload {
			resource := resources[index]
			switch kind, _ := resourceKind(resource); kind {
			case inventory.KindVM:
				return s.scanVM(ctx, resource.Node, resource)
			case inventory.KindContainer:
				return s.scanContainer(ctx, resource.Node, resource)
			default:
				return inventory.Workload{}
			}
		}),
	}
	return snapshot, true
}

func (s *Scanner) scanNode(ctx context.Context, node string, snapshot *inventory.Snapshot) {
	vms, err := s.api.VirtualMachines(ctx, node)
	if err != nil {
		snapshot.Problems = append(snapshot.Problems, problem(node, inventory.KindVM, 0, "list", err))
	} else {
		snapshot.Workloads = append(snapshot.Workloads, s.scanVMs(ctx, node, vms)...)
	}

	containers, err := s.api.Containers(ctx, node)
	if err != nil {
		snapshot.Problems = append(snapshot.Problems, problem(node, inventory.KindContainer, 0, "list", err))
		return
	}

	snapshot.Workloads = append(snapshot.Workloads, s.scanContainers(ctx, node, containers)...)
}

func (s *Scanner) scanVMs(ctx context.Context, node string, resources []proxmox.Resource) []inventory.Workload {
	vms := s.filteredResources(resources)
	return s.scanWorkloads(len(vms), func(index int) inventory.Workload {
		return s.scanVM(ctx, node, vms[index])
	})
}

func (s *Scanner) scanContainers(ctx context.Context, node string, resources []proxmox.Resource) []inventory.Workload {
	containers := s.filteredResources(resources)
	return s.scanWorkloads(len(containers), func(index int) inventory.Workload {
		return s.scanContainer(ctx, node, containers[index])
	})
}

func (s *Scanner) filteredClusterResources(resources []proxmox.Resource) []proxmox.Resource {
	filtered := make([]proxmox.Resource, 0, len(resources))
	for _, resource := range resources {
		if _, ok := resourceKind(resource); !ok {
			continue
		}
		if !s.includedNode(resource.Node) {
			continue
		}
		if s.includeResource(resource) {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

func (s *Scanner) filteredResources(resources []proxmox.Resource) []proxmox.Resource {
	filtered := make([]proxmox.Resource, 0, len(resources))
	for _, resource := range resources {
		if s.includeResource(resource) {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

func (s *Scanner) includeResource(resource proxmox.Resource) bool {
	if s.options.SkipStopped && resource.Status != "running" {
		return false
	}
	return s.matchesRequiredTags(splitTags(resource.Tags))
}

func (s *Scanner) scanWorkloads(count int, scan func(index int) inventory.Workload) []inventory.Workload {
	if count == 0 {
		return nil
	}

	// Use explicit loops and WaitGroup.Add/Done for Yaegi plugin compatibility.
	workloads := make([]inventory.Workload, count)
	if count == 1 || s.maxConcurrency <= 1 {
		for index := 0; index < count; index++ {
			workloads[index] = scan(index)
		}
		return workloads
	}

	limit := minInt(s.maxConcurrency, count)

	var wg sync.WaitGroup
	sem := make(chan struct{}, limit)
	for index := 0; index < count; index++ {
		index := index
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			workloads[index] = scan(index)
		}()
	}
	wg.Wait()
	return workloads
}

func (s *Scanner) scanVM(ctx context.Context, node string, resource proxmox.Resource) inventory.Workload {
	workload := baseWorkload(inventory.KindVM, node, resource)

	cfg, err := s.api.VMConfig(ctx, node, resource.VMID)
	if err != nil {
		workload.Problems = append(workload.Problems, problem(node, inventory.KindVM, resource.VMID, "config", err))
		return workload
	}
	s.applyConfig(&workload, cfg)

	return workload
}

func (s *Scanner) scanContainer(ctx context.Context, node string, resource proxmox.Resource) inventory.Workload {
	workload := baseWorkload(inventory.KindContainer, node, resource)

	cfg, err := s.api.ContainerConfig(ctx, node, resource.VMID)
	if err != nil {
		workload.Problems = append(workload.Problems, problem(node, inventory.KindContainer, resource.VMID, "config", err))
		return workload
	}
	s.applyConfig(&workload, cfg)

	return workload
}

func (s *Scanner) applyConfig(workload *inventory.Workload, cfg proxmox.GuestConfig) {
	if cfg.Name != "" {
		workload.Name = cfg.Name
	} else if cfg.Hostname != "" {
		workload.Name = cfg.Hostname
	}
	workload.Notes = cfg.Description
	if cfg.Tags != "" {
		workload.Tags = splitTags(cfg.Tags)
	}
}

func baseWorkload(kind inventory.Kind, node string, resource proxmox.Resource) inventory.Workload {
	return inventory.Workload{
		Kind:   kind,
		Node:   node,
		ID:     resource.VMID,
		Name:   resource.Name,
		Status: resource.Status,
		Tags:   splitTags(resource.Tags),
	}
}

func resourceKind(resource proxmox.Resource) (inventory.Kind, bool) {
	switch resource.Type {
	case "qemu":
		return inventory.KindVM, true
	case "lxc":
		return inventory.KindContainer, true
	default:
		return "", false
	}
}

func clusterResourcesUsable(resources []proxmox.Resource) bool {
	for _, resource := range resources {
		if _, ok := resourceKind(resource); !ok {
			return false
		}
		if strings.TrimSpace(resource.Node) == "" || resource.VMID == 0 || strings.TrimSpace(resource.Name) == "" || strings.TrimSpace(resource.Status) == "" {
			return false
		}
	}
	return true
}

func compareClusterResources(a, b proxmox.Resource) int {
	if a.Node != b.Node {
		if a.Node < b.Node {
			return -1
		}
		return 1
	}

	if rank := resourceKindRank(a) - resourceKindRank(b); rank != 0 {
		return rank
	}

	return a.VMID - b.VMID
}

func resourceKindRank(resource proxmox.Resource) int {
	switch resource.Type {
	case "qemu":
		return 0
	case "lxc":
		return 1
	default:
		return 2
	}
}

func problem(node string, kind inventory.Kind, id int, stage string, err error) inventory.Problem {
	return inventory.Problem{
		Node:    node,
		Kind:    kind,
		ID:      id,
		Stage:   stage,
		Message: err.Error(),
	}
}
