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
	if len(cfg.PVE.DefaultInterfaces) != 4 ||
		cfg.PVE.DefaultInterfaces[0] != "eth*" ||
		cfg.PVE.DefaultInterfaces[1] != "enp*" ||
		cfg.PVE.DefaultInterfaces[2] != "eno*" ||
		cfg.PVE.DefaultInterfaces[3] != "ens*" {
		t.Fatalf("PVE.DefaultInterfaces = %#v", cfg.PVE.DefaultInterfaces)
	}
	if cfg.PVE.MaxConcurrency != 4 {
		t.Fatalf("PVE.MaxConcurrency = %d", cfg.PVE.MaxConcurrency)
	}
}
