package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/F0903/traefik-pve-provider/metadata"
	"github.com/F0903/traefik-pve-provider/proxmox"
)

const defaultMaxConcurrency = 4

type ProxmoxAPI interface {
	Nodes(ctx context.Context) ([]proxmox.Node, error)
	VirtualMachines(ctx context.Context, node string) ([]proxmox.Resource, error)
	Containers(ctx context.Context, node string) ([]proxmox.Resource, error)
	VMConfig(ctx context.Context, node string, vmid int) (proxmox.GuestConfig, error)
	ContainerConfig(ctx context.Context, node string, vmid int) (proxmox.GuestConfig, error)
	VMNetworkInterfaces(ctx context.Context, node string, vmid int) (proxmox.GuestAgentInterfaces, error)
	ContainerInterfaces(ctx context.Context, node string, vmid int) ([]proxmox.NetworkInterface, error)
}

type ScanOptions struct {
	SkipStopped      bool
	SkipIPResolution bool
	MetadataMode     metadata.Mode
	Nodes            []string
	RequiredTags     []string
	MaxConcurrency   int
}

type Scanner struct {
	api               ProxmoxAPI
	parser            metadata.Parser
	options           ScanOptions
	includedNodeNames []string
	includedNodes     map[string]bool
	requiredTags      []string
	maxConcurrency    int
}

func NewScanner(api ProxmoxAPI, options ScanOptions) *Scanner {
	return &Scanner{
		api:               api,
		parser:            metadata.Parser{Prefix: metadata.DefaultPrefix, Mode: options.MetadataMode},
		options:           options,
		includedNodeNames: normalizedNodeNames(options.Nodes),
		includedNodes:     normalizedSet(options.Nodes),
		requiredTags:      normalizedList(options.RequiredTags),
		maxConcurrency:    normalizedMaxConcurrency(options.MaxConcurrency),
	}
}

func (s *Scanner) Scan(ctx context.Context) (Snapshot, error) {
	if len(s.includedNodeNames) > 0 {
		snapshot := Snapshot{}
		for _, node := range s.includedNodeNames {
			s.scanNode(ctx, node, &snapshot)
		}
		return snapshot, nil
	}

	nodes, err := s.api.Nodes(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("list nodes: %w", err)
	}

	snapshot := Snapshot{}
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

func (s *Scanner) scanNode(ctx context.Context, node string, snapshot *Snapshot) {
	vms, err := s.api.VirtualMachines(ctx, node)
	if err != nil {
		snapshot.Problems = append(snapshot.Problems, problem(node, KindVM, 0, "list", err))
	} else {
		snapshot.Workloads = append(snapshot.Workloads, s.scanVMs(ctx, node, vms)...)
	}

	containers, err := s.api.Containers(ctx, node)
	if err != nil {
		snapshot.Problems = append(snapshot.Problems, problem(node, KindContainer, 0, "list", err))
		return
	}

	snapshot.Workloads = append(snapshot.Workloads, s.scanContainers(ctx, node, containers)...)
}

func (s *Scanner) scanVMs(ctx context.Context, node string, resources []proxmox.Resource) []Workload {
	vms := s.filteredResources(resources)
	return s.scanWorkloads(len(vms), func(index int) Workload {
		return s.scanVM(ctx, node, vms[index])
	})
}

func (s *Scanner) scanContainers(ctx context.Context, node string, resources []proxmox.Resource) []Workload {
	containers := s.filteredResources(resources)
	return s.scanWorkloads(len(containers), func(index int) Workload {
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

func (s *Scanner) scanWorkloads(count int, scan func(index int) Workload) []Workload {
	if count == 0 {
		return nil
	}

	workloads := make([]Workload, count)
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

func (s *Scanner) scanVM(ctx context.Context, node string, resource proxmox.Resource) Workload {
	workload := baseWorkload(KindVM, node, resource)

	cfg, err := s.api.VMConfig(ctx, node, resource.VMID)
	if err != nil {
		workload.Problems = append(workload.Problems, problem(node, KindVM, resource.VMID, "config", err))
		return workload
	}
	s.applyConfig(&workload, cfg)

	if s.shouldResolveIPs(workload, resource.Status) {
		interfaces, err := s.api.VMNetworkInterfaces(ctx, node, resource.VMID)
		if err != nil {
			workload.Problems = append(workload.Problems, interfaceProblem(node, KindVM, resource.VMID, err))
		} else {
			workload.IPs = ipsFromInterfaces(interfaces.Result)
		}
	}

	return workload
}

func (s *Scanner) scanContainer(ctx context.Context, node string, resource proxmox.Resource) Workload {
	workload := baseWorkload(KindContainer, node, resource)

	cfg, err := s.api.ContainerConfig(ctx, node, resource.VMID)
	if err != nil {
		workload.Problems = append(workload.Problems, problem(node, KindContainer, resource.VMID, "config", err))
		return workload
	}
	s.applyConfig(&workload, cfg)

	if s.shouldResolveIPs(workload, resource.Status) {
		interfaces, err := s.api.ContainerInterfaces(ctx, node, resource.VMID)
		if err != nil {
			workload.Problems = append(workload.Problems, interfaceProblem(node, KindContainer, resource.VMID, err))
		} else {
			workload.IPs = ipsFromInterfaces(interfaces)
		}
	}

	return workload
}

func (s *Scanner) applyConfig(workload *Workload, cfg proxmox.GuestConfig) {
	if cfg.Name != "" {
		workload.Name = cfg.Name
	} else if cfg.Hostname != "" {
		workload.Name = cfg.Hostname
	}
	workload.Notes = cfg.Description
	if cfg.Tags != "" {
		workload.Tags = splitTags(cfg.Tags)
	}

	parsed := s.parser.Parse(cfg.Description)
	workload.TraefikLabels = parsed.Labels
	workload.LabelDiagnostics = parsed.Diagnostics
}

func (s *Scanner) shouldResolveIPs(workload Workload, status string) bool {
	return !s.options.SkipIPResolution && status == "running" && labelsEnableTraefik(workload.TraefikLabels)
}

func (s *Scanner) includedNode(node string) bool {
	if len(s.includedNodes) == 0 {
		return true
	}
	return s.includedNodes[strings.ToLower(strings.TrimSpace(node))]
}

func (s *Scanner) matchesRequiredTags(tags []string) bool {
	if len(s.requiredTags) == 0 {
		return true
	}

	tagSet := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tagSet[strings.ToLower(tag)] = true
	}
	for _, required := range s.requiredTags {
		if !tagSet[required] {
			return false
		}
	}
	return true
}

func baseWorkload(kind Kind, node string, resource proxmox.Resource) Workload {
	return Workload{
		Kind:          kind,
		Node:          node,
		ID:            resource.VMID,
		Name:          resource.Name,
		Status:        resource.Status,
		Tags:          splitTags(resource.Tags),
		TraefikLabels: make(map[string]string),
	}
}

func splitTags(tags string) []string {
	if tags == "" {
		return nil
	}

	raw := strings.FieldsFunc(tags, func(r rune) bool {
		return r == ';' || r == ',' || r == ' '
	})
	result := make([]string, 0, len(raw))
	for _, tag := range raw {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			result = append(result, tag)
		}
	}
	return result
}

func normalizedNodeNames(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}

func normalizedSet(values []string) map[string]bool {
	list := normalizedList(values)
	if len(list) == 0 {
		return nil
	}
	set := make(map[string]bool, len(list))
	for _, value := range list {
		set[value] = true
	}
	return set
}

func normalizedList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func normalizedMaxConcurrency(value int) int {
	if value <= 0 {
		return defaultMaxConcurrency
	}
	return value
}

func labelsEnableTraefik(labels map[string]string) bool {
	enabled, ok := parseBool(labels["traefik.enable"])
	return ok && enabled
}

func parseBool(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "on":
		return true, true
	case "false", "0", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func problem(node string, kind Kind, id int, stage string, err error) Problem {
	return Problem{
		Node:    node,
		Kind:    kind,
		ID:      id,
		Stage:   stage,
		Message: err.Error(),
	}
}

func interfaceProblem(node string, kind Kind, id int, err error) Problem {
	message := err.Error()
	var apiErr *proxmox.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == 403 {
		message += "; check that the Proxmox API token has interface discovery privileges such as VM.GuestAgent.Audit on PVE 9+"
	}
	return Problem{
		Node:    node,
		Kind:    kind,
		ID:      id,
		Stage:   "interfaces",
		Message: message,
	}
}
