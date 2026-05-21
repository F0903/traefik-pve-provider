package scanner

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/F0903/traefik-pve-provider/proxmox"
	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
)

const defaultMaxConcurrency = 4

var defaultInterfacePatterns = []string{"eth*", "enp*", "eno*"}

type IPMode string

const (
	IPModeIPv4      IPMode = "ipv4"
	IPModeIPv6      IPMode = "ipv6"
	IPModeIPv4IPv6  IPMode = "ipv4/6"
	IPModeDualStack IPMode = "dualstack"
)

var ErrInvalidIPMode = errors.New("invalid IP mode")

// Needed instead of min for Yaegi compatibility.
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
	SkipStopped       bool
	SkipIPResolution  bool
	IPMode            IPMode
	DefaultInterfaces []string
	Nodes             []string
	RequiredTags      []string
	MaxConcurrency    int
}

type Scanner struct {
	api               ProxmoxAPI
	options           Options
	includedNodeNames []string
	includedNodes     map[string]bool
	requiredTags      []string
	maxConcurrency    int
	ipMode            IPMode
	defaultInterfaces []string
}

func New(api ProxmoxAPI, options Options) *Scanner {
	return &Scanner{
		api:               api,
		options:           options,
		includedNodeNames: normalizedNodeNames(options.Nodes),
		includedNodes:     normalizedSet(options.Nodes),
		requiredTags:      normalizedList(options.RequiredTags),
		maxConcurrency:    normalizedMaxConcurrency(options.MaxConcurrency),
		ipMode:            normalizedIPMode(options.IPMode),
		defaultInterfaces: normalizedDefaultInterfaces(options.DefaultInterfaces),
	}
}

func DefaultInterfacePatterns() []string {
	return cloneStringSlice(defaultInterfacePatterns)
}

func (s *Scanner) DefaultInterfaces() []string {
	return cloneStringSlice(s.defaultInterfaces)
}

func normalizedDefaultInterfaces(patterns []string) []string {
	if patterns == nil {
		return DefaultInterfacePatterns()
	}
	return normalizedInterfacePatterns(patterns)
}

func cloneStringSlice(values []string) []string {
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func ParseIPMode(raw string) (IPMode, error) {
	if raw == "" {
		return IPModeIPv4, nil
	}

	switch mode := IPMode(strings.ToLower(strings.TrimSpace(raw))); mode {
	case IPModeIPv4, "4":
		return IPModeIPv4, nil
	case IPModeIPv6, "6":
		return IPModeIPv6, nil
	case IPModeIPv4IPv6, IPModeDualStack, "dual-stack", "both", "all", "ipv4+ipv6", "ipv4,ipv6":
		return IPModeIPv4IPv6, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidIPMode, raw)
	}
}

func normalizedIPMode(mode IPMode) IPMode {
	parsed, err := ParseIPMode(string(mode))
	if err != nil {
		return IPModeIPv4
	}
	return parsed
}

func (m IPMode) allows(version int) bool {
	switch normalizedIPMode(m) {
	case IPModeIPv6:
		return version == 6
	case IPModeIPv4IPv6:
		return version == 4 || version == 6
	default:
		return version == 4
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
