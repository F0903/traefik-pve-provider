package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/F0903/traefik-pve-provider/metadata"
	"github.com/F0903/traefik-pve-provider/proxmox"
)

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
}

type Scanner struct {
	api     ProxmoxAPI
	parser  metadata.Parser
	options ScanOptions
}

func NewScanner(api ProxmoxAPI, options ScanOptions) *Scanner {
	return &Scanner{
		api:     api,
		parser:  metadata.Parser{Prefix: metadata.DefaultPrefix, Mode: options.MetadataMode},
		options: options,
	}
}

func (s *Scanner) Scan(ctx context.Context) (Snapshot, error) {
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
		for _, vm := range vms {
			if s.options.SkipStopped && vm.Status != "running" {
				continue
			}
			workload := s.scanVM(ctx, node, vm)
			if s.matchesRequiredTags(workload.Tags) {
				snapshot.Workloads = append(snapshot.Workloads, workload)
			}
		}
	}

	containers, err := s.api.Containers(ctx, node)
	if err != nil {
		snapshot.Problems = append(snapshot.Problems, problem(node, KindContainer, 0, "list", err))
		return
	}

	for _, container := range containers {
		if s.options.SkipStopped && container.Status != "running" {
			continue
		}
		workload := s.scanContainer(ctx, node, container)
		if s.matchesRequiredTags(workload.Tags) {
			snapshot.Workloads = append(snapshot.Workloads, workload)
		}
	}
}

func (s *Scanner) scanVM(ctx context.Context, node string, resource proxmox.Resource) Workload {
	workload := baseWorkload(KindVM, node, resource)

	cfg, err := s.api.VMConfig(ctx, node, resource.VMID)
	if err != nil {
		workload.Problems = append(workload.Problems, problem(node, KindVM, resource.VMID, "config", err))
		return workload
	}
	s.applyConfig(&workload, cfg)

	if !s.options.SkipIPResolution && resource.Status == "running" {
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

	if !s.options.SkipIPResolution && resource.Status == "running" {
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

func (s *Scanner) includedNode(node string) bool {
	if len(s.options.Nodes) == 0 {
		return true
	}
	for _, included := range s.options.Nodes {
		if strings.EqualFold(strings.TrimSpace(included), node) {
			return true
		}
	}
	return false
}

func (s *Scanner) matchesRequiredTags(tags []string) bool {
	if len(s.options.RequiredTags) == 0 {
		return true
	}

	tagSet := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tagSet[strings.ToLower(tag)] = true
	}
	for _, required := range s.options.RequiredTags {
		required = strings.ToLower(strings.TrimSpace(required))
		if required == "" {
			continue
		}
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
