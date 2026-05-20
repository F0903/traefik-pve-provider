//go:build yaegi

package traefik_pve_provider

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const yaegiVersion = "v0.16.1"

func TestYaegiImportsPlugin(t *testing.T) {
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tempDir := t.TempDir()
	gopath := filepath.Join(tempDir, "gopath")
	t.Cleanup(func() {
		if err := makeWritable(filepath.Join(gopath, "pkg", "mod")); err != nil {
			t.Logf("make module cache writable: %v", err)
		}
	})

	pluginPath := filepath.Join(gopath, "src", "github.com", "F0903", "traefik-pve-provider")
	if err := copyDir(repoRoot, pluginPath); err != nil {
		t.Fatalf("copy plugin source: %v", err)
	}

	harnessDir := filepath.Join(tempDir, "harness")
	if err := os.MkdirAll(harnessDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(harnessDir, "go.mod"), []byte(yaegiHarnessGoMod()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(harnessDir, "main.go"), []byte(yaegiHarnessMain()), 0o644); err != nil {
		t.Fatal(err)
	}

	env := commandEnv(map[string]string{
		"GOFLAGS": "-mod=mod",
		"GOPATH":  gopath,
	})
	runTestCommand(t, harnessDir, env, "go", "mod", "tidy")
	runTestCommand(t, harnessDir, env, "go", "run", ".")
}

func yaegiHarnessGoMod() string {
	return fmt.Sprintf(`module yaegiimporttest

go 1.19

require github.com/traefik/yaegi %s
`, yaegiVersion)
}

func yaegiHarnessMain() string {
	const eval = `import "encoding/json"
import pve "github.com/F0903/traefik-pve-provider"
import proxmox "github.com/F0903/traefik-pve-provider/proxmox"
import inventory "github.com/F0903/traefik-pve-provider/proxmox/inventory"
import tconfig "github.com/F0903/traefik-pve-provider/traefik"
import labels "github.com/F0903/traefik-pve-provider/traefik/labels"

var _ = pve.CreateConfig()
var _ = func() bool {
	parsed, diagnostics := labels.Parse(map[string]string{"traefik.enable": "true", "traefik.port": "8080"}, "app")
	if len(diagnostics) != 0 {
		panic("unexpected diagnostics")
	}
	if !parsed.Enabled() {
		panic("labels disabled")
	}
	var guestConfig proxmox.GuestConfig
	if err := json.Unmarshal([]byte("{\"name\":\"nas\",\"description\":\"notes\",\"ipconfig0\":\"ip=10.0.0.50/24\"}"), &guestConfig); err != nil {
		panic(err)
	}
	if guestConfig.Name != "nas" || guestConfig.Description != "notes" {
		panic("guest config did not decode")
	}

	fence := string([]byte{96, 96, 96})
	prepared := tconfig.Prepare(inventory.Snapshot{
		Workloads: []inventory.Workload{
			{
				Kind:   inventory.KindVM,
				Node:   "pve1",
				ID:     100,
				Name:   "app",
				Status: "running",
				IPs:    []inventory.IP{{Address: "10.0.0.10", Version: 4}},
				Notes:  fence + "traefik\nenable=true\nport=8080\n" + fence,
			},
		},
	}, tconfig.PrepareOptions{})
	if len(prepared.Workloads) != 1 || !prepared.Workloads[0].Labels.Enabled() {
		panic("workload labels were not prepared")
	}

	config := tconfig.BuildPreparedConfiguration(prepared, tconfig.Options{DefaultDomain: "example.com"})
	router := config.HTTP.Routers["app"]
	if router == nil || router.Service != "app" {
		panic("missing HTTP router")
	}
	service := config.HTTP.Services["app"]
	if service == nil || service.LoadBalancer == nil || len(service.LoadBalancer.Servers) != 1 {
		panic("missing HTTP service")
	}
	if service.LoadBalancer.Servers[0].URL != "http://10.0.0.10:8080" {
		panic("unexpected HTTP server URL")
	}
	if payload, err := tconfig.Marshal(config); err != nil || len(payload) == 0 {
		panic("configuration did not marshal")
	}
	return true
}()`
	return fmt.Sprintf(`package main

import (
	"fmt"
	"os"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

func main() {
	i := interp.New(interp.Options{GoPath: os.Getenv("GOPATH")})
	if err := i.Use(stdlib.Symbols); err != nil {
		panic(err)
	}
	if _, err := i.Eval(%q); err != nil {
		panic(err)
	}
	fmt.Println("yaegi import ok")
}
`, eval)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == src {
			return os.MkdirAll(dst, 0o755)
		}

		name := entry.Name()
		if entry.IsDir() && name == ".git" {
			return filepath.SkipDir
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func makeWritable(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if entry.IsDir() {
			return os.Chmod(path, 0o755)
		}
		return os.Chmod(path, 0o644)
	})
}

func runTestCommand(t *testing.T, dir string, env []string, name string, args ...string) {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", strings.Join(cmd.Args, " "), err, output)
	}
}

func commandEnv(overrides map[string]string) []string {
	env := make([]string, 0, len(os.Environ())+len(overrides))
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if _, override := overrides[key]; override {
			continue
		}
		env = append(env, value)
	}
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	return env
}
