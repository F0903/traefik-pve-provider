package scanner

import (
	"context"
	"fmt"

	"github.com/F0903/traefik-pve-provider/proxmox"
	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
)

const defaultMaxConcurrency = 4

// Needed intstead of min for Yeagi compatibility
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type ProxmoxAPI interface {
	Nodes(ctx context.Context) ([]proxmox.Node, error)
	ClusterResources(ctx context.Context) ([]proxmox.Resource, error)
	VirtualMachines(ctx context.Context, node string) ([]proxmox.Resource, error)
	Containers(ctx context.Context, node string) ([]proxmox.Resource, error)
	VMConfig(ctx context.Context, node string, vmid int) (proxmox.GuestConfig, error)
	ContainerConfig(ctx context.Context, node string, vmid int) (proxmox.GuestConfig, error)
	VMNetworkInterfaces(ctx context.Context, node string, vmid int) (proxmox.GuestAgentInterfaces, error)
	ContainerInterfaces(ctx context.Context, node string, vmid int) ([]proxmox.NetworkInterface, error)
}

type Options struct {
	SkipStopped      bool
	SkipIPResolution bool
	Nodes            []string
	RequiredTags     []string
	MaxConcurrency   int
}

type Scanner struct {
	api               ProxmoxAPI
	options           Options
	includedNodeNames []string
	includedNodes     map[string]bool
	requiredTags      []string
	maxConcurrency    int
}

func New(api ProxmoxAPI, options Options) *Scanner {
	return &Scanner{
		api:               api,
		options:           options,
		includedNodeNames: normalizedNodeNames(options.Nodes),
		includedNodes:     normalizedSet(options.Nodes),
		requiredTags:      normalizedList(options.RequiredTags),
		maxConcurrency:    normalizedMaxConcurrency(options.MaxConcurrency),
	}
}

func (s *Scanner) Scan(ctx context.Context) (inventory.Snapshot, error) {
	if snapshot, ok := s.scanClusterResources(ctx); ok {
		return snapshot, nil
	}
	return s.scanNodes(ctx)
}

func (s *Scanner) scanNodes(ctx context.Context) (inventory.Snapshot, error) {
	if len(s.includedNodeNames) > 0 {
		snapshot := inventory.Snapshot{}
		for _, node := range s.includedNodeNames {
			s.scanNode(ctx, node, &snapshot)
		}
		return snapshot, nil
	}

	nodes, err := s.api.Nodes(ctx)
	if err != nil {
		return inventory.Snapshot{}, fmt.Errorf("list nodes: %w", err)
	}

	snapshot := inventory.Snapshot{}
	for _, node := range nodes {
		if node.Status != "" && node.Status != "online" {
			continue
		}
		if !s.includedNode(node.Name) {
			continue
		}
		s.scanNode(ctx, node.Name, &snapshot)
	}

	return snapshot, nil
}
