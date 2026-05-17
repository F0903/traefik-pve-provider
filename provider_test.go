package traefik_pve_provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/F0903/traefik-pve-provider/proxmox/inventory"
	"github.com/F0903/traefik-pve-provider/traefik"
)

func TestNewAcceptsCatalogTestDataShape(t *testing.T) {
	provider, err := New(context.Background(), &Config{
		PollInterval: "60s",
		PVE: PVEConfig{
			Endpoint:           "https://pve.example.com",
			TokenID:            "root@pam!traefik",
			Token:              "00000000-0000-0000-0000-000000000000",
			Timeout:            "5s",
			InsecureSkipVerify: true,
			SkipStopped:        true,
			SkipIPResolution:   true,
			MaxConcurrency:     4,
		},
		MetadataMode:  "fenced",
		DefaultDomain: "example.com",
	}, "traefik-pve-provider")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if provider.pollInterval != 60*time.Second {
		t.Fatalf("poll interval = %s", provider.pollInterval)
	}
	if provider.configOptions.DefaultDomain != "example.com" {
		t.Fatalf("default domain = %q", provider.configOptions.DefaultDomain)
	}
}

func TestNewRejectsTooSmallPollInterval(t *testing.T) {
	_, err := New(context.Background(), &Config{
		PollInterval: "1s",
		PVE: PVEConfig{
			Endpoint: "https://pve.example.com",
			TokenID:  "root@pam!traefik",
			Token:    "secret",
		},
	}, "traefik-pve-provider")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewRejectsInvalidMetadataMode(t *testing.T) {
	_, err := New(context.Background(), &Config{
		PollInterval: "60s",
		PVE: PVEConfig{
			Endpoint: "https://pve.example.com",
			TokenID:  "root@pam!traefik",
			Token:    "secret",
		},
		MetadataMode: "unknown",
	}, "traefik-pve-provider")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPublishSkipsUnchangedConfiguration(t *testing.T) {
	provider := &Provider{
		scanner: stubScanner{snapshot: inventory.Snapshot{}},
	}
	cfgChan := make(chan json.Marshaler, 2)

	provider.publish(context.Background(), cfgChan)
	if len(cfgChan) != 1 {
		t.Fatalf("published config count = %d, want 1", len(cfgChan))
	}

	provider.publish(context.Background(), cfgChan)
	if len(cfgChan) != 1 {
		t.Fatalf("published config count = %d, want unchanged count 1", len(cfgChan))
	}
}

func TestPublishSkipsScanError(t *testing.T) {
	provider := &Provider{
		scanner: stubScanner{err: errors.New("scan failed")},
	}
	cfgChan := make(chan json.Marshaler, 1)

	provider.publish(context.Background(), cfgChan)
	if len(cfgChan) != 0 {
		t.Fatalf("published config count = %d, want 0", len(cfgChan))
	}
}

func TestPublishConfigurationReturnsWhenContextCanceled(t *testing.T) {
	provider := &Provider{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfgChan := make(chan json.Marshaler)
	err := provider.publishConfiguration(ctx, traefik.BuildConfiguration(inventory.Snapshot{}), cfgChan)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("publishConfiguration() error = %v, want context.Canceled", err)
	}
	if len(provider.lastPayload) != 0 {
		t.Fatalf("lastPayload was updated after canceled publish")
	}
}

func TestPublishResolvesIPsOnlyForEnabledRunningWorkloads(t *testing.T) {
	scanner := &recordingScanner{
		snapshot: inventory.Snapshot{
			Workloads: []inventory.Workload{
				{ID: 100, Name: "enabled", Status: "running", Notes: "```traefik\nenable=true\n```"},
				{ID: 101, Name: "disabled", Status: "running", Notes: "```traefik\nenable=false\n```"},
				{ID: 102, Name: "stopped", Status: "stopped", Notes: "```traefik\nenable=true\n```"},
			},
		},
	}
	provider := &Provider{scanner: scanner}
	cfgChan := make(chan json.Marshaler, 1)

	provider.publish(context.Background(), cfgChan)

	if len(scanner.resolved) != 1 || scanner.resolved[0].ID != 100 {
		t.Fatalf("resolved workloads = %#v", scanner.resolved)
	}
}

func TestProblemLogMessagesIncludesScanLabelAndConfigDiagnostics(t *testing.T) {
	snapshot := traefik.Prepare(inventory.Snapshot{
		Problems: []inventory.Problem{
			{Node: "pve1", Kind: inventory.KindVM, Stage: "list", Message: "failed"},
		},
		Workloads: []inventory.Workload{
			{
				Node:  "pve1",
				Kind:  inventory.KindContainer,
				ID:    100,
				Notes: "```traefik\nbad\n```",
				Problems: []inventory.Problem{
					{Node: "pve1", Kind: inventory.KindContainer, ID: 100, Stage: "interfaces", Message: "forbidden"},
				},
			},
		},
	}, traefik.PrepareOptions{})

	messages := problemLogMessages(snapshot, []traefik.Diagnostic{
		{Node: "pve1", Kind: inventory.KindContainer, ID: 100, Message: "unsupported label", Line: 4, Fragment: "port=eight"},
	})

	if len(messages) != 4 {
		t.Fatalf("messages = %#v", messages)
	}
	if !messagesContain(messages, "labels: missing '=' in label (line 2: bad)") {
		t.Fatalf("missing label source context: %#v", messages)
	}
	if !messagesContain(messages, "config: unsupported label (line 4: port=eight)") {
		t.Fatalf("missing config source context: %#v", messages)
	}
}

func messagesContain(messages []string, fragment string) bool {
	for _, message := range messages {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

type stubScanner struct {
	snapshot inventory.Snapshot
	err      error
}

func (s stubScanner) Scan(ctx context.Context) (inventory.Snapshot, error) {
	return s.snapshot, s.err
}

func (s stubScanner) ResolveIPs(ctx context.Context, workloads []*inventory.Workload) {}

type recordingScanner struct {
	snapshot inventory.Snapshot
	resolved []*inventory.Workload
}

func (s *recordingScanner) Scan(ctx context.Context) (inventory.Snapshot, error) {
	return s.snapshot, nil
}

func (s *recordingScanner) ResolveIPs(ctx context.Context, workloads []*inventory.Workload) {
	s.resolved = append(s.resolved, workloads...)
}
