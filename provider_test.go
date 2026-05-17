package traefik_pve_provider

import (
	"context"
	"encoding/json"
	"errors"
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

func TestProblemLogMessagesIncludesScanLabelAndConfigDiagnostics(t *testing.T) {
	messages := problemLogMessages(inventory.Snapshot{
		Problems: []inventory.Problem{
			{Node: "pve1", Kind: inventory.KindVM, Stage: "list", Message: "failed"},
		},
		Workloads: []inventory.Workload{
			{
				Node: "pve1",
				Kind: inventory.KindContainer,
				ID:   100,
				Problems: []inventory.Problem{
					{Node: "pve1", Kind: inventory.KindContainer, ID: 100, Stage: "interfaces", Message: "forbidden"},
				},
			},
		},
	}, []traefik.Diagnostic{
		{Node: "pve1", Kind: inventory.KindContainer, ID: 100, Message: "unsupported label"},
	})

	if len(messages) != 3 {
		t.Fatalf("messages = %#v", messages)
	}
}

type stubScanner struct {
	snapshot inventory.Snapshot
	err      error
}

func (s stubScanner) Scan(ctx context.Context) (inventory.Snapshot, error) {
	return s.snapshot, s.err
}
