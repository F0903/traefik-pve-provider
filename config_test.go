package traefik_pve_provider

import "testing"

func TestCreateConfig(t *testing.T) {
	cfg := CreateConfig()

	if cfg.PollInterval != "1m0s" {
		t.Fatalf("PollInterval = %q", cfg.PollInterval)
	}
	if cfg.PVE.Timeout != "30s" {
		t.Fatalf("PVE.Timeout = %q", cfg.PVE.Timeout)
	}
	if cfg.MetadataMode != "fenced" {
		t.Fatalf("MetadataMode = %q", cfg.MetadataMode)
	}
	if !cfg.PVE.SkipStopped {
		t.Fatal("PVE.SkipStopped = false")
	}
	if cfg.PVE.IPMode != "ipv4" {
		t.Fatalf("PVE.IPMode = %q", cfg.PVE.IPMode)
	}
	if cfg.PVE.MaxConcurrency != 4 {
		t.Fatalf("PVE.MaxConcurrency = %d", cfg.PVE.MaxConcurrency)
	}
}
