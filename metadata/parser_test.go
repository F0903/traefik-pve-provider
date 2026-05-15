package metadata

import (
	"errors"
	"testing"
)

func TestParseNotesFencedTraefikBlock(t *testing.T) {
	result := ParseNotes("notes before\n```traefik\n# exposed through Traefik\nenable=true\n\nhttp.routers.app.rule=Host(`app.example.com`)\n```\nnotes after")

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

func TestParseNotesDefaultIgnoresLooseLabels(t *testing.T) {
	result := ParseNotes("traefik.enable=true\ntraefik.http.routers.app.rule=Host(`app.example.com`)")

	if len(result.Labels) != 0 {
		t.Fatalf("labels = %#v, want empty", result.Labels)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want empty", result.Diagnostics)
	}
}

func TestParseNotesDefaultIgnoresPrefixlessLooseLabels(t *testing.T) {
	result := ParseNotes("enable=true\nport=8080")

	if len(result.Labels) != 0 {
		t.Fatalf("labels = %#v, want empty", result.Labels)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want empty", result.Diagnostics)
	}
}

func TestParseNotesIgnoresOtherCodeFences(t *testing.T) {
	result := ParseNotes("```yaml\ntraefik.enable=true\n```\n```traefik\ntraefik.enable=false\n```")

	if got := result.Labels["traefik.enable"]; got != "false" {
		t.Fatalf("traefik.enable = %q, want false", got)
	}
}

func TestParseNotesFencedReportsMalformedAndDuplicateLabels(t *testing.T) {
	result := ParseNotes("```traefik\ntraefik.enable\ntraefik.enable=false\ntraefik.enable=true\n```\n")

	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q, want true", got)
	}
	if len(result.Diagnostics) != 2 {
		t.Fatalf("diagnostic count = %d, want 2: %#v", len(result.Diagnostics), result.Diagnostics)
	}
}

func TestParseNotesFencedAcceptsPrefixlessKeys(t *testing.T) {
	result := ParseNotes("```traefik\nenable=true\nport=8080\nmiddlewares=local-only@file,compress-all@file\n```")

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

func TestParseNotesFencedTreatsPrefixedAndPrefixlessKeysAsDuplicates(t *testing.T) {
	result := ParseNotes("```traefik\nenable=false\ntraefik.enable=true\n```")

	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q, want true", got)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want duplicate diagnostic", result.Diagnostics)
	}
}

func TestParseNotesFencedRejectsInvalidPrefixlessKeys(t *testing.T) {
	result := ParseNotes("```traefik\ninvalid key=true\n```")

	if len(result.Labels) != 0 {
		t.Fatalf("labels = %#v, want empty", result.Labels)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want invalid key diagnostic", result.Diagnostics)
	}
}

func TestParseNotesFencedReportsUnterminatedBlock(t *testing.T) {
	result := ParseNotes("```traefik\ntraefik.enable=true")

	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q, want true", got)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one diagnostic", result.Diagnostics)
	}
}

func TestParseNotesLooseModeWhitespaceSeparatedLabels(t *testing.T) {
	result := Parser{Mode: ModeLoose}.Parse("traefik.enable=true traefik.http.routers.app.entrypoints=websecure traefik.http.services.app.loadbalancer.server.port=8080")

	if len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", result.Diagnostics)
	}
	if len(result.Labels) != 3 {
		t.Fatalf("label count = %d, want 3", len(result.Labels))
	}
	if got := result.Labels["traefik.http.services.app.loadbalancer.server.port"]; got != "8080" {
		t.Fatalf("port = %q, want 8080", got)
	}
}

func TestParseNotesLooseModeKeepsSpacesInsideValues(t *testing.T) {
	result := Parser{Mode: ModeLoose}.Parse("traefik.http.routers.app.rule=Host(`app.example.com`) && PathPrefix(`/api`) traefik.enable=true")

	if got := result.Labels["traefik.http.routers.app.rule"]; got != "Host(`app.example.com`) && PathPrefix(`/api`)" {
		t.Fatalf("rule = %q", got)
	}
	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q", got)
	}
}

func TestParseNotesNormalizesKeysAndPreservesValues(t *testing.T) {
	result := ParseNotes("```traefik\nTraefik.HTTP.Services.App.LoadBalancer.ServersTransport=secureTransport@file\n```")

	if got := result.Labels["traefik.http.services.app.loadbalancer.serverstransport"]; got != "secureTransport@file" {
		t.Fatalf("servers transport = %q", got)
	}
}

func TestParseNotesAutoFallsBackToLooseLabels(t *testing.T) {
	result := Parser{Mode: ModeAuto}.Parse("traefik.enable=true\ntraefik.http.routers.app.rule=Host(`app.example.com`)")

	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q, want true", got)
	}
	if got := result.Labels["traefik.http.routers.app.rule"]; got != "Host(`app.example.com`)" {
		t.Fatalf("rule = %q", got)
	}
}

func TestParseNotesAutoPrefersFencedLabels(t *testing.T) {
	result := Parser{Mode: ModeAuto}.Parse("traefik.enable=false\n```traefik\ntraefik.enable=true\n```")

	if got := result.Labels["traefik.enable"]; got != "true" {
		t.Fatalf("traefik.enable = %q, want true", got)
	}
}

func TestParseMode(t *testing.T) {
	mode, err := ParseMode("")
	if err != nil {
		t.Fatalf("ParseMode() error = %v", err)
	}
	if mode != ModeFenced {
		t.Fatalf("mode = %q, want %q", mode, ModeFenced)
	}

	mode, err = ParseMode("AUTO")
	if err != nil {
		t.Fatalf("ParseMode() error = %v", err)
	}
	if mode != ModeAuto {
		t.Fatalf("mode = %q, want %q", mode, ModeAuto)
	}

	_, err = ParseMode("unknown")
	if !errors.Is(err, ErrInvalidMode) {
		t.Fatalf("error = %v, want ErrInvalidMode", err)
	}
}
