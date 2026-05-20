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
	const eval = `import pve "github.com/F0903/traefik-pve-provider"; var _ = pve.CreateConfig()`
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
