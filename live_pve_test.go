package traefik_pve_provider

import (
	"context"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/F0903/traefik-pve-provider/proxmox"
	inventoryscanner "github.com/F0903/traefik-pve-provider/proxmox/inventory/scanner"
	"github.com/F0903/traefik-pve-provider/traefik"
	"github.com/F0903/traefik-pve-provider/traefik/labels"
)

func TestLivePVEParsesConfiguredGuests(t *testing.T) {
	if os.Getenv("PVE_LIVE_TEST") != "1" {
		t.Skip("set PVE_LIVE_TEST=1 to run this read-only live PVE integration test")
	}

	env := livePVEEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := proxmox.NewClient(proxmox.Config{
		Endpoint:           env["PVE_HOST"],
		TokenID:            env["PVE_TOKEN_ID"],
		Token:              env["PVE_SECRET"],
		Timeout:            30 * time.Second,
		InsecureSkipVerify: liveBool(env["PVE_INSECURE_SKIP_VERIFY"], true),
		UserAgent:          "traefik-pve-provider-live-test",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	scanner := inventoryscanner.New(client, inventoryscanner.Options{
		SkipIPResolution: true,
	})
	snapshot, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	prepared := traefik.Prepare(snapshot, traefik.PrepareOptions{
		ExtractMode: labels.ExtractModeAuto,
	})
	result := traefik.BuildPrepared(prepared, traefik.Options{
		DefaultDomain: env["PVE_TEST_DEFAULT_DOMAIN"],
	})

	totalLabels := 0
	configuredWorkloads := 0
	enabledWorkloads := 0
	extractDiagnostics := 0
	parseDiagnostics := 0

	for _, workload := range prepared.Workloads {
		labelCount := len(workload.Labels.Raw)
		totalLabels += labelCount
		extractDiagnostics += len(workload.Labels.ExtractDiagnostics)
		parseDiagnostics += len(workload.Labels.ParseDiagnostics)

		if labelCount == 0 && len(workload.Labels.ExtractDiagnostics) == 0 && len(workload.Labels.ParseDiagnostics) == 0 {
			continue
		}

		configuredWorkloads++
		if workload.Labels.Enabled() {
			enabledWorkloads++
		}

		t.Logf(
			"guest node=%s kind=%s id=%d name=%q status=%s labels=%d enabled=%t extractDiagnostics=%d parseDiagnostics=%d configProblems=%d",
			workload.Node,
			workload.Kind,
			workload.ID,
			workload.Name,
			workload.Status,
			labelCount,
			workload.Labels.Enabled(),
			len(workload.Labels.ExtractDiagnostics),
			len(workload.Labels.ParseDiagnostics),
			len(workload.Problems),
		)
		if labelCount > 0 {
			t.Logf("guest node=%s kind=%s id=%d labelKeys=%s", workload.Node, workload.Kind, workload.ID, strings.Join(sortedLiveLabelKeys(workload.Labels.Raw), ","))
		}
		for _, diagnostic := range workload.Labels.ExtractDiagnostics {
			t.Logf("guest node=%s kind=%s id=%d extractDiagnostic=line %d fragment=%q message=%q", workload.Node, workload.Kind, workload.ID, diagnostic.Line, diagnostic.Fragment, diagnostic.Message)
		}
		for _, diagnostic := range workload.Labels.ParseDiagnostics {
			t.Logf("guest node=%s kind=%s id=%d parseDiagnostic=line %d fragment=%q key=%q value=%q error=%q", workload.Node, workload.Kind, workload.ID, diagnostic.Source.Line, diagnostic.Source.Fragment, diagnostic.Key, diagnostic.Value, diagnostic.Err)
		}
	}

	t.Logf(
		"summary workloads=%d configured=%d enabled=%d labels=%d scanProblems=%d extractDiagnostics=%d parseDiagnostics=%d configDiagnostics=%d",
		len(prepared.Workloads),
		configuredWorkloads,
		enabledWorkloads,
		totalLabels,
		len(snapshot.Problems),
		extractDiagnostics,
		parseDiagnostics,
		len(result.Diagnostics),
	)
	t.Logf(
		"config httpRouters=%d httpServices=%d tcpRouters=%d tcpServices=%d udpRouters=%d udpServices=%d",
		len(result.Configuration.HTTP.Routers),
		len(result.Configuration.HTTP.Services),
		len(result.Configuration.TCP.Routers),
		len(result.Configuration.TCP.Services),
		len(result.Configuration.UDP.Routers),
		len(result.Configuration.UDP.Services),
	)
	for _, diagnostic := range result.Diagnostics {
		t.Logf("configDiagnostic node=%s kind=%s id=%d line=%d fragment=%q message=%q", diagnostic.Node, diagnostic.Kind, diagnostic.ID, diagnostic.Line, diagnostic.Fragment, diagnostic.Message)
	}

	if totalLabels == 0 {
		t.Fatal("no Traefik labels were extracted from live PVE notes")
	}
}

func livePVEEnv(t *testing.T) map[string]string {
	t.Helper()

	env := map[string]string{
		"PVE_HOST":                 os.Getenv("PVE_HOST"),
		"PVE_TOKEN_ID":             os.Getenv("PVE_TOKEN_ID"),
		"PVE_SECRET":               os.Getenv("PVE_SECRET"),
		"PVE_INSECURE_SKIP_VERIFY": os.Getenv("PVE_INSECURE_SKIP_VERIFY"),
		"PVE_TEST_DEFAULT_DOMAIN":  os.Getenv("PVE_TEST_DEFAULT_DOMAIN"),
	}

	if values, err := readLiveDotEnv(".env"); err == nil {
		for key, value := range values {
			if env[key] == "" {
				env[key] = value
			}
		}
	}

	for _, key := range []string{"PVE_HOST", "PVE_TOKEN_ID", "PVE_SECRET"} {
		if strings.TrimSpace(env[key]) == "" {
			t.Fatalf("%s is required for live PVE test", key)
		}
	}
	return env
}

func readLiveDotEnv(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	values := make(map[string]string)
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" {
			values[key] = value
		}
	}
	return values, nil
}

func liveBool(value string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return fallback
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return fallback
	}
}

func sortedLiveLabelKeys(labels map[string]string) []string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
