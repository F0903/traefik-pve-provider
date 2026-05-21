package labels

import (
	"errors"
	"testing"
)

func TestExtractFencedTraefikBlock(t *testing.T) {
	result := Extract("notes before\n```traefik\n# exposed through Traefik\nenable=true\n\nhttp.routers.app.rule=Host(`app.example.com`)\n```\nnotes after")

	if len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", result.Diagnostics)
	}
	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q, want true", got)
	}
	if got := result.Labels["traefik.http.routers.app.rule"]; got != "Host(`app.example.com`)" {
		t.Fatalf("router rule = %q", got)
	}
}

func TestExtractDefaultIgnoresLooseLabels(t *testing.T) {
	result := Extract("traefik.enable=true\ntraefik.http.routers.app.rule=Host(`app.example.com`)")

	if len(result.Labels) != 0 {
		t.Fatalf("labels = %#v, want empty", result.Labels)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want empty", result.Diagnostics)
	}
}

func TestExtractDefaultIgnoresPrefixlessLooseLabels(t *testing.T) {
	result := Extract("enable=true\nport=8080")

	if len(result.Labels) != 0 {
		t.Fatalf("labels = %#v, want empty", result.Labels)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want empty", result.Diagnostics)
	}
}

func TestExtractIgnoresOtherCodeFences(t *testing.T) {
	result := Extract("```yaml\ntraefik.enable=true\n```\n```traefik\ntraefik.enable=false\n```")

	if got := result.Labels["traefik.enable"]; got != "false" {
		t.Fatalf("traefik.enable = %q, want false", got)
	}
}

func TestExtractFencedReportsMalformedAndDuplicateLabels(t *testing.T) {
	result := Extract("```traefik\ntraefik.enable\ntraefik.enable=false\ntraefik.enable=true\n```\n")

	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q, want true", got)
	}
	if len(result.Diagnostics) != 2 {
		t.Fatalf("diagnostic count = %d, want 2: %#v", len(result.Diagnostics), result.Diagnostics)
	}
	if result.Diagnostics[0].Line != 2 || result.Diagnostics[0].Fragment != "traefik.enable" {
		t.Fatalf("first diagnostic source = %#v", result.Diagnostics[0])
	}
	if source := result.Sources["traefik.enable"]; source.Line != 4 || source.Fragment != "traefik.enable=true" {
		t.Fatalf("traefik.enable source = %#v", source)
	}
}

func TestExtractFencedParsesOnlyFirstTraefikBlock(t *testing.T) {
	result := Extract("```traefik\nenable=false\n```\n```traefik\nenable=true\nbad\n```")

	if got := result.Labels["traefik.enable"]; got != "false" {
		t.Fatalf("traefik.enable = %q, want false", got)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want empty", result.Diagnostics)
	}
}

func TestExtractFencedAcceptsPrefixlessKeys(t *testing.T) {
	result := Extract("```traefik\nenable=true\nport=8080\nmiddlewares=local-only@file,compress-all@file\n```")

	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want empty", result.Diagnostics)
	}
	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q, want true", got)
	}
	if got := result.Labels["traefik.port"]; got != "8080" {
		t.Fatalf("traefik.port = %q, want 8080", got)
	}
	if got := result.Labels["traefik.middlewares"]; got != "local-only@file,compress-all@file" {
		t.Fatalf("traefik.middlewares = %q", got)
	}
}

func TestExtractFencedAcceptsProviderMetadataLabels(t *testing.T) {
	result := Extract("```traefik\nenable=true\npve.interfaces=eth*,ens18\n```")

	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want empty", result.Diagnostics)
	}
	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q, want true", got)
	}
	if got := result.Labels["pve.interfaces"]; got != "eth*,ens18" {
		t.Fatalf("pve.interfaces = %q", got)
	}
}

func TestExtractFencedTreatsPrefixedAndPrefixlessKeysAsDuplicates(t *testing.T) {
	result := Extract("```traefik\nenable=false\ntraefik.enable=true\n```")

	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q, want true", got)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want duplicate diagnostic", result.Diagnostics)
	}
}

func TestExtractFencedRejectsInvalidPrefixlessKeys(t *testing.T) {
	result := Extract("```traefik\ninvalid key=true\n```")

	if len(result.Labels) != 0 {
		t.Fatalf("labels = %#v, want empty", result.Labels)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want invalid key diagnostic", result.Diagnostics)
	}
}

func TestExtractFencedReportsUnterminatedBlock(t *testing.T) {
	result := Extract("```traefik\ntraefik.enable=true")

	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q, want true", got)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one diagnostic", result.Diagnostics)
	}
}

func TestExtractLooseModeWhitespaceSeparatedLabels(t *testing.T) {
	result := Extractor{Mode: ExtractModeLoose}.Extract("traefik.enable=true pve.interfaces=eth* traefik.http.routers.app.entrypoints=websecure traefik.http.services.app.loadbalancer.server.port=8080")

	if len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", result.Diagnostics)
	}
	if len(result.Labels) != 4 {
		t.Fatalf("label count = %d, want 4", len(result.Labels))
	}
	if got := result.Labels["pve.interfaces"]; got != "eth*" {
		t.Fatalf("pve.interfaces = %q, want eth*", got)
	}
	if got := result.Labels["traefik.http.services.app.loadbalancer.server.port"]; got != "8080" {
		t.Fatalf("port = %q, want 8080", got)
	}
}

func TestExtractLooseModeKeepsSpacesInsideValues(t *testing.T) {
	result := Extractor{Mode: ExtractModeLoose}.Extract("traefik.http.routers.app.rule=Host(`app.example.com`) && PathPrefix(`/api`) traefik.enable=true")

	if got := result.Labels["traefik.http.routers.app.rule"]; got != "Host(`app.example.com`) && PathPrefix(`/api`)" {
		t.Fatalf("rule = %q", got)
	}
	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q", got)
	}
}

func TestExtractNormalizesKeysAndPreservesValues(t *testing.T) {
	result := Extract("```traefik\nTraefik.HTTP.Services.App.LoadBalancer.ServersTransport=secureTransport@file\n```")

	if got := result.Labels["traefik.http.services.app.loadbalancer.serverstransport"]; got != "secureTransport@file" {
		t.Fatalf("servers transport = %q", got)
	}
}

func TestExtractSupportsCustomPrefix(t *testing.T) {
	result := Extractor{Prefix: "custom.", Mode: ExtractModeLoose}.Extract("custom.enable=true\ntraefik.enable=false")

	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want empty", result.Diagnostics)
	}
	if got := result.Labels["custom.enable"]; got != "true" {
		t.Fatalf("custom.enable = %q, want true", got)
	}
	if _, exists := result.Labels["traefik.enable"]; exists {
		t.Fatalf("traefik.enable was extracted with custom prefix: %#v", result.Labels)
	}
}

func TestExtractAutoFallsBackToLooseLabels(t *testing.T) {
	result := Extractor{Mode: ExtractModeAuto}.Extract("traefik.enable=true\ntraefik.http.routers.app.rule=Host(`app.example.com`)")

	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q, want true", got)
	}
	if got := result.Labels["traefik.http.routers.app.rule"]; got != "Host(`app.example.com`)" {
		t.Fatalf("rule = %q", got)
	}
}

func TestExtractAutoPrefersFencedLabels(t *testing.T) {
	result := Extractor{Mode: ExtractModeAuto}.Extract("traefik.enable=false\n```traefik\ntraefik.enable=true\n```")

	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q, want true", got)
	}
}

func TestParseExtractMode(t *testing.T) {
	mode, err := ParseExtractMode("")
	if err != nil {
		t.Fatalf("ParseExtractMode() error = %v", err)
	}
	if mode != ExtractModeFenced {
		t.Fatalf("mode = %q, want %q", mode, ExtractModeFenced)
	}

	mode, err = ParseExtractMode("AUTO")
	if err != nil {
		t.Fatalf("ParseExtractMode() error = %v", err)
	}
	if mode != ExtractModeAuto {
		t.Fatalf("mode = %q, want %q", mode, ExtractModeAuto)
	}

	_, err = ParseExtractMode("unknown")
	if !errors.Is(err, ErrInvalidMode) {
		t.Fatalf("error = %v, want ErrInvalidMode", err)
	}
}
