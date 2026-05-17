package scanner

import (
	"context"
	"sync"

	"github.com/F0903/traefik-pve-provider/proxmox"
	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
)

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

func (s *Scanner) filteredResources(resources []proxmox.Resource) []proxmox.Resource {
	filtered := make([]proxmox.Resource, 0, len(resources))
	for _, resource := range resources {
		if s.options.SkipStopped && resource.Status != "running" {
			continue
		}
		if !s.matchesRequiredTags(splitTags(resource.Tags)) {
			continue
		}
		filtered = append(filtered, resource)
	}
	return filtered
}

func (s *Scanner) scanWorkloads(count int, scan func(index int) inventory.Workload) []inventory.Workload {
	if count == 0 {
		return nil
	}

	workloads := make([]inventory.Workload, count)
	if count == 1 || s.maxConcurrency <= 1 {
		for index := range count {
			workloads[index] = scan(index)
		}
		return workloads
	}

	limit := min(s.maxConcurrency, count)

	var wg sync.WaitGroup
	sem := make(chan struct{}, limit)
	for index := range count {
		sem <- struct{}{}
		wg.Go(func() {
			defer func() { <-sem }()
			workloads[index] = scan(index)
		})
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

func problem(node string, kind inventory.Kind, id int, stage string, err error) inventory.Problem {
	return inventory.Problem{
		Node:    node,
		Kind:    kind,
		ID:      id,
		Stage:   stage,
		Message: err.Error(),
	}
}
