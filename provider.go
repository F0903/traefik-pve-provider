package traefik_pve_provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/F0903/traefik-pve-provider/proxmox"
	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	inventoryscanner "github.com/F0903/traefik-pve-provider/proxmox/inventory/scanner"
	"github.com/F0903/traefik-pve-provider/traefik"
	"github.com/F0903/traefik-pve-provider/traefik/labels"
	"github.com/traefik/genconf/dynamic"
)

type Provider struct {
	name           string
	pollInterval   time.Duration
	scanner        scanner
	cancel         context.CancelFunc
	lastPayload    []byte
	configOptions  traefik.Options
	prepareOptions traefik.PrepareOptions
	lastProblems   map[string]bool
}

type scanner interface {
	Scan(ctx context.Context) (inventory.Snapshot, error)
	ResolveIPs(ctx context.Context, workloads []*inventory.Workload)
}

func New(ctx context.Context, config *Config, name string) (*Provider, error) {
	_ = ctx

	if config == nil {
		return nil, errors.New("configuration cannot be nil")
	}

	pollInterval, err := parseDuration(config.PollInterval, defaultPollInterval, "poll interval")
	if err != nil {
		return nil, err
	}
	if pollInterval < minPollInterval {
		return nil, fmt.Errorf("poll interval must be at least %s", minPollInterval)
	}

	pveConfig := config.PVE

	timeout, err := parseDuration(pveConfig.Timeout, defaultPVETimeout, "pve timeout")
	if err != nil {
		return nil, err
	}

	ipMode, err := inventoryscanner.ParseIPMode(pveConfig.IPMode)
	if err != nil {
		return nil, err
	}

	extractMode, err := labels.ParseExtractMode(config.MetadataMode)
	if err != nil {
		return nil, err
	}

	client, err := proxmox.NewClient(proxmox.Config{
		Endpoint:           pveConfig.Endpoint,
		TokenID:            pveConfig.TokenID,
		Token:              pveConfig.Token,
		Timeout:            timeout,
		InsecureSkipVerify: pveConfig.InsecureSkipVerify,
		UserAgent:          "traefik-pve-provider/" + name,
	})
	if err != nil {
		return nil, err
	}

	return &Provider{
		name:         name,
		pollInterval: pollInterval,
		scanner: inventoryscanner.New(client, inventoryscanner.Options{
			SkipStopped:      pveConfig.SkipStopped,
			SkipIPResolution: pveConfig.SkipIPResolution,
			IPMode:           ipMode,
			Nodes:            pveConfig.Nodes,
			RequiredTags:     pveConfig.RequiredTags,
			MaxConcurrency:   pveConfig.MaxConcurrency,
		}),
		configOptions: traefik.Options{
			DefaultDomain: config.DefaultDomain,
		},
		prepareOptions: traefik.PrepareOptions{
			ExtractMode: extractMode,
		},
	}, nil
}

func (p *Provider) Init() error {
	return nil
}

func (p *Provider) Provide(cfgChan chan<- json.Marshaler) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	go p.run(ctx, cfgChan)
	return nil
}

func (p *Provider) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func (p *Provider) run(ctx context.Context, cfgChan chan<- json.Marshaler) {
	p.publish(ctx, cfgChan)

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.publish(ctx, cfgChan)
		case <-ctx.Done():
			return
		}
	}
}

func (p *Provider) publish(ctx context.Context, cfgChan chan<- json.Marshaler) {
	snapshot, err := p.scanner.Scan(ctx)
	if err != nil {
		log.Printf("traefik-pve-provider: scan failed: %v", err)
		return
	}

	prepared := traefik.Prepare(snapshot, p.prepareOptions)
	p.scanner.ResolveIPs(ctx, prepared.EnabledRunningWorkloads())

	result := traefik.BuildPrepared(prepared, p.configOptions)
	p.logProblems(prepared, result.Diagnostics)
	if err := p.publishConfiguration(ctx, result.Configuration, cfgChan); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("traefik-pve-provider: failed to publish configuration: %v", err)
	}
}

func (p *Provider) publishConfiguration(ctx context.Context, configuration *dynamic.Configuration, cfgChan chan<- json.Marshaler) error {
	payload, err := traefik.Marshal(configuration)
	if err != nil {
		return fmt.Errorf("marshal configuration: %w", err)
	}
	if bytes.Equal(p.lastPayload, payload) {
		return nil
	}

	select {
	case cfgChan <- json.RawMessage(payload):
	case <-ctx.Done():
		return ctx.Err()
	}

	p.lastPayload = append(p.lastPayload[:0], payload...)
	return nil
}

func (p *Provider) logProblems(snapshot traefik.PreparedSnapshot, diagnostics []traefik.Diagnostic) {
	messages := problemLogMessages(snapshot, diagnostics)
	current := make(map[string]bool, len(messages))
	for _, message := range messages {
		current[message] = true
		if !p.lastProblems[message] {
			log.Print(message)
		}
	}
	p.lastProblems = current
}

func problemLogMessages(snapshot traefik.PreparedSnapshot, diagnostics []traefik.Diagnostic) []string {
	messages := make([]string, 0)
	for _, problem := range snapshot.Problems {
		messages = append(messages, fmt.Sprintf("traefik-pve-provider: node=%s kind=%s stage=%s: %s", problem.Node, problem.Kind, problem.Stage, problem.Message))
	}
	for _, workload := range snapshot.Workloads {
		for _, problem := range workload.Problems {
			messages = append(messages, fmt.Sprintf("traefik-pve-provider: node=%s kind=%s id=%d stage=%s: %s", problem.Node, problem.Kind, problem.ID, problem.Stage, problem.Message))
		}
		for _, diagnostic := range workload.Labels.ExtractDiagnostics {
			messages = append(messages, fmt.Sprintf("traefik-pve-provider: node=%s kind=%s id=%d labels: %s", workload.Node, workload.Kind, workload.ID, diagnosticMessage(diagnostic.Message, diagnostic.Line, diagnostic.Fragment)))
		}
	}
	for _, diagnostic := range diagnostics {
		messages = append(messages, fmt.Sprintf("traefik-pve-provider: node=%s kind=%s id=%d config: %s", diagnostic.Node, diagnostic.Kind, diagnostic.ID, diagnosticMessage(diagnostic.Message, diagnostic.Line, diagnostic.Fragment)))
	}
	sort.Strings(messages)
	return messages
}

func diagnosticMessage(message string, line int, fragment string) string {
	switch {
	case line > 0 && fragment != "":
		return fmt.Sprintf("%s (line %d: %s)", message, line, fragment)
	case line > 0:
		return fmt.Sprintf("%s (line %d)", message, line)
	case fragment != "":
		return fmt.Sprintf("%s (fragment: %s)", message, fragment)
	default:
		return message
	}
}
