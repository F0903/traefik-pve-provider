package traefik_pve_provider

import (
	"fmt"
	"time"

	"github.com/F0903/traefik-pve-provider/metadata"
)

const (
	defaultPollInterval = 60 * time.Second
	defaultPVETimeout   = 30 * time.Second
	minPollInterval     = 5 * time.Second
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
	Nodes              []string `json:"nodes,omitempty" yaml:"nodes,omitempty" toml:"nodes,omitempty"`
	RequiredTags       []string `json:"requiredTags,omitempty" yaml:"requiredTags,omitempty" toml:"requiredTags,omitempty"`
}

func CreateConfig() *Config {
	return &Config{
		PollInterval: defaultPollInterval.String(),
		MetadataMode: string(metadata.ModeFenced),
		PVE: PVEConfig{
			Timeout:     defaultPVETimeout.String(),
			SkipStopped: true,
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
