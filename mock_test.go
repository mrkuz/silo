package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// cmdCall records a single execCommand invocation.
type cmdCall struct {
	name string
	args []string
}

// mockExecCommand installs a fake execCommand that records every call and returns
// preset *exec.Cmd values keyed by the full command string (e.g. "podman image exists foo").
// Calls with no preset entry return exec.Command("true") (exit 0 silently).
// It also registers a Cleanup that restores the original execCommand.
// Returns the recorder function (for installation) and a pointer to the recorded calls.
func mockExecCommand(t *testing.T, responses map[string]*exec.Cmd) *[]cmdCall {
	t.Helper()
	calls := &[]cmdCall{}
	orig := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		*calls = append(*calls, cmdCall{name: name, args: args})
		key := strings.Join(append([]string{name}, args...), " ")
		if cmd, ok := responses[key]; ok {
			return cmd
		}
		return exec.Command("true")
	}
	t.Cleanup(func() { execCommand = orig })
	return calls
}

// anyCall reports whether calls contains any entry whose joined string
// contains all the provided substrings.
func anyCall(calls *[]cmdCall, substrings ...string) bool {
	for _, c := range *calls {
		joined := strings.Join(append([]string{c.name}, c.args...), " ")
		match := true
		for _, sub := range substrings {
			if !strings.Contains(joined, sub) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// setupWorkspace creates a temp directory, writes a .silo/silo.toml from cfg,
// and os.Chdir into it. The original directory is restored via t.Cleanup.
// NOTE: os.Chdir is process-global — do not use t.Parallel() in tests calling this.
func setupWorkspace(t *testing.T, cfg Config) string {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".silo"), 0755); err != nil {
		t.Fatalf("mkdir .silo: %v", err)
	}
	f, err := os.Create(filepath.Join(dir, ".silo", "silo.toml"))
	if err != nil {
		t.Fatalf("create silo.toml: %v", err)
	}
	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		t.Fatalf("encode silo.toml: %v", err)
	}
	f.Close()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	return dir
}

// setupGlobalConfig points XDG_CONFIG_HOME at a new temp directory and writes
// the minimal files required by ensureScaffoldFiles and buildBaseImage.
// Needed by any test that calls initWorkspaceConfig or ensureImages.
func setupGlobalConfig(t *testing.T) {
	t.Helper()
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	siloDir := filepath.Join(base, "silo")
	if err := os.MkdirAll(siloDir, 0755); err != nil {
		t.Fatalf("mkdir silo config dir: %v", err)
	}
	// home.nix is read by buildBaseImage; write the minimal empty module.
	if err := os.WriteFile(filepath.Join(siloDir, "home.nix"), []byte(emptyHomeNix), 0644); err != nil {
		t.Fatalf("write home.nix: %v", err)
	}
	if err := os.WriteFile(filepath.Join(siloDir, "silo.toml"), []byte{}, 0644); err != nil {
		t.Fatalf("write silo.toml: %v", err)
	}
}

// minimalConfig returns a Config suitable for use in unit tests.
func minimalConfig(id string) Config {
	return Config{
		General: GeneralConfig{
			ID:            id,
			User:          "testuser",
			ContainerName: "silo-" + id,
			ImageName:     "silo-" + id,
		},
		Features:     FeaturesConfig{Workspace: false, SharedVolume: false, Nested: false},
		SharedVolume: SharedVolumeConfig{Paths: []string{}},
		Connect:      ConnectConfig{Command: "/bin/sh"},
		Create:       CreateConfig{ExtraArgs: []string{}},
	}
}
