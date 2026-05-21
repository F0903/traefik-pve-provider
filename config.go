package traefik_pve_provider

import (
	"fmt"
	"time"

	inventoryscanner "github.com/F0903/traefik-pve-provider/proxmox/inventory/scanner"
	"github.com/F0903/traefik-pve-provider/traefik/labels"
)

const (
	defaultPollInterval      = 60 * time.Second
	defaultPVETimeout        = 30 * time.Second
	defaultPVEMaxConcurrency = 4
	defaultPVEIPMode         = "ipv4"
	minPollInterval          = 5 * time.Second
)

type Config struct {
	PollInterval  string    `json:"pollInterval,omitempty" yaml:"pollInterval,omitempty" toml:"pollInterval,omitempty"`
	PVE           PVEConfig `json:"pve" yaml:"pve,omitempty" toml:"pve,omitempty"`
	MetadataMode  string    `json:"metadataMode,omitempty" yaml:"metadataMode,omitempty" toml:"metadataMode,omitempty"`
	DefaultDomain string    `json:"defaultDomain,omitempty" yaml:"defaultDomain,omitempty" toml:"defaultDomain,omitempty"`
}

type PVEConfig struct {
	Endpoint           string   `json:"endpoint,omitempty" yaml:"endpoint,omitempty" toml:"endpoint,omitempty"`
	TokenID            string   `json:"tokenID,omitempty" yaml:"tokenID,omitempty" toml:"tokenID,omitempty"`
	Token              string   `json:"token,omitempty" yaml:"token,omitempty" toml:"token,omitempty"`
	Timeout            string   `json:"timeout,omitempty" yaml:"timeout,omitempty" toml:"timeout,omitempty"`
	InsecureSkipVerify bool     `json:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty" toml:"insecureSkipVerify,omitempty"`
	SkipStopped        bool     `json:"skipStopped,omitempty" yaml:"skipStopped,omitempty" toml:"skipStopped,omitempty"`
	SkipIPResolution   bool     `json:"skipIPResolution,omitempty" yaml:"skipIPResolution,omitempty" toml:"skipIPResolution,omitempty"`
	IPMode             string   `json:"ipMode,omitempty" yaml:"ipMode,omitempty" toml:"ipMode,omitempty"`
	DefaultInterfaces  []string `json:"defaultInterfaces,omitempty" yaml:"defaultInterfaces,omitempty" toml:"defaultInterfaces,omitempty"`
	Nodes              []string `json:"nodes,omitempty" yaml:"nodes,omitempty" toml:"nodes,omitempty"`
	RequiredTags       []string `json:"requiredTags,omitempty" yaml:"requiredTags,omitempty" toml:"requiredTags,omitempty"`
	MaxConcurrency     int      `json:"maxConcurrency,omitempty" yaml:"maxConcurrency,omitempty" toml:"maxConcurrency,omitempty"`
}

func CreateConfig() *Config {
	return &Config{
		PollInterval: defaultPollInterval.String(),
		MetadataMode: string(labels.ExtractModeFenced),
		PVE: PVEConfig{
			Timeout:           defaultPVETimeout.String(),
			SkipStopped:       true,
			IPMode:            defaultPVEIPMode,
			DefaultInterfaces: inventoryscanner.DefaultInterfacePatterns(),
			MaxConcurrency:    defaultPVEMaxConcurrency,
		},
	}
}

func parseDuration(raw string, fallback time.Duration, field string) (time.Duration, error) {
	if raw == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", field, err)
	}
	return duration, nil
}
