package internal

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// SetGeneratedIDFunc overrides the ID generation function for tests.
// Call it with nil to reset to the default.
func SetGeneratedIDFunc(t *testing.T, fn func() string) {
	t.Helper()
	if fn == nil {
		generatedIDFunc = generateID
		return
	}
	generatedIDFunc = fn
}

// FirstRun sets up a first-run scenario: fresh XDG_CONFIG_HOME, empty workspace dir,
// and mocked execCommand. Returns the XDG_CONFIG_HOME base path.
func FirstRun(t *testing.T) string {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	return base
}

// FirstRunWith sets up a first-run scenario with a pre-populated user config directory.
// The configFunc receives the Silo config directory path for customization.
// Returns the XDG_CONFIG_HOME base path for use when the caller's ft.Base is needed.
func FirstRunWith(t *testing.T, configFunc func(siloUser string)) string {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	siloUser := base + "/silo"
	if err := os.MkdirAll(siloUser, 0755); err != nil {
		t.Fatal(err)
	}
	if configFunc != nil {
		configFunc(siloUser)
	}

	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	return base
}

// FirstRunWithFiles sets up a first-run scenario with user config files written.
// starterFiles maps filename to content (e.g., "home.user.nix" -> content).
// Returns the XDG_CONFIG_HOME base path.
func FirstRunWithFiles(t *testing.T, starterFiles map[string]string) string {
	return FirstRunWith(t, func(siloUser string) {
		for name, content := range starterFiles {
			WriteUserFile(t, siloUser, name, content)
		}
	})
}

// SubsequentRun sets up an existing workspace with config cfg and calls SetupUserConfig
// for user-level files. Returns the XDG_CONFIG_HOME path.
func SubsequentRun(t *testing.T, cfg Config) string {
	SetupWorkspace(t, cfg)
	SetupUserConfig(t)
	return os.Getenv("XDG_CONFIG_HOME")
}

// CaptureStdout runs fn with stdout redirected to a buffer and returns the output.
// It restores stdout after fn completes (even if it panics).
func CaptureStdout(fn func()) string {
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = stdout
	return buf.String()
}

// CaptureStderr runs fn with stderr redirected to a buffer and returns the output.
// It restores stderr after fn completes (even if it panics).
func CaptureStderr(fn func()) string {
	r, w, _ := os.Pipe()
	stderr := os.Stderr
	os.Stderr = w
	fn()
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stderr = stderr
	return buf.String()
}

// WriteUserFile writes content to a file under the user's silo config directory.
// It creates the parent directory if needed and calls t.Fatal on error.
func WriteUserFile(t *testing.T, siloUser, name, content string) {
	t.Helper()
	path := filepath.Join(siloUser, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// WriteUserToml encodes cfg as TOML and writes it to a file under the user's silo config directory.
func WriteUserToml(t *testing.T, siloUser, name string, cfg Config) {
	t.Helper()
	path := filepath.Join(siloUser, name)
	if err := WriteTOML(path, cfg); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// SetupWorkspace creates a temp directory, writes a .silo/silo.toml from cfg,
// and os.Chdir into it. The original directory is restored via t.Cleanup.
// NOTE: os.Chdir is process-global — do not use t.Parallel() in tests calling this.
func SetupWorkspace(t *testing.T, cfg Config) string {
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
	f.Close()
	if err := WriteTOML(filepath.Join(dir, ".silo", "silo.toml"), cfg); err != nil {
		t.Fatalf("write silo.toml: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	return dir
}

// SetupUserConfig points XDG_CONFIG_HOME at a new temp directory and writes
// the minimal files required by EnsureUserFiles and BuildUserImage.
// Needed by any test that calls InitWorkspaceConfig or EnsureImages.
func SetupUserConfig(t *testing.T) {
	t.Helper()
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	siloDir := filepath.Join(base, "silo")
	if err := os.MkdirAll(siloDir, 0755); err != nil {
		t.Fatalf("mkdir silo config dir: %v", err)
	}
	// home.user.nix is read by BuildUserImage; write the minimal empty module.
	if err := os.WriteFile(filepath.Join(siloDir, "home.user.nix"), []byte("{\n  config,\n  pkgs,\n  ...\n}:\n{\n}\n"), 0644); err != nil {
		t.Fatalf("write home.user.nix: %v", err)
	}
	if err := os.WriteFile(filepath.Join(siloDir, "silo.in.toml"), []byte{}, 0644); err != nil {
		t.Fatalf("write silo.in.toml: %v", err)
	}
}

// MinimalConfig returns a Config suitable for use in unit tests.
func MinimalConfig(id string) Config {
	return Config{
		General:      GeneralConfig{ID: id, User: "testuser"},
		Features:     FeaturesConfig{Podman: false},
		SharedVolume: SharedVolumeConfig{Paths: []string{}},
		Podman:       PodmanConfig{CreateArgs: []string{}},
	}
}