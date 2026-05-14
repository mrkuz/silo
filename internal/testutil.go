package internal

import (
	"bytes"
	"fmt"
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

// SetupWorkspace sets up an existing workspace with config cfg and user,
// and calls SetupUserFiles for user-level files.
// Returns the XDG_CONFIG_HOME path.
func SetupWorkspace(t *testing.T, cfg WorkspaceConfig, user string) string {
	SetupWorkspaceFiles(t, cfg)
	SetupUserFiles(t, user)
	return os.Getenv("XDG_CONFIG_HOME")
}

// SetupMinimalWorkspace sets up an existing workspace with a minimal config using id,
// and calls SetupUserFiles for user-level files.
// Returns the XDG_CONFIG_HOME path.
func SetupMinimalWorkspace(t *testing.T, id, user string) string {
	return SetupWorkspace(t, MinimalWorkspaceConfig(id), user)
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

// WriteUserFile writes content to a file under the user's silo config directory.
// It creates the parent directory if needed and calls t.Fatal on error.
func WriteUserFile(t *testing.T, siloUser, name, content string) {
	t.Helper()
	path := filepath.Join(siloUser, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// SetupWorkspaceFiles creates a temp directory, writes a .silo/silo.toml from cfg,
// and os.Chdir into it. The original directory is restored via t.Cleanup.
// NOTE: os.Chdir is process-global — do not use t.Parallel() in tests calling this.
func SetupWorkspaceFiles(t *testing.T, cfg WorkspaceConfig) string {
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

// SetupEmptyWorkspace creates a temp directory, chdirs into it, and restores
// the original directory on cleanup. No .silo directory is created.
// NOTE: os.Chdir is process-global — do not use t.Parallel() in tests calling this.
func SetupEmptyWorkspace(t *testing.T) string {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	return dir
}

// SetupUserFiles points XDG_CONFIG_HOME at a new temp directory and writes
// the minimal files required by EnsureUserFiles and BuildUserImage.
// Needed by any test that calls EnsureInit or EnsureBuild.
func SetupUserFiles(t *testing.T, users ...string) {
	t.Helper()
	user := "testuser"
	if len(users) > 0 {
		user = users[0]
	}
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	siloDir := filepath.Join(base, "silo")
	if err := os.MkdirAll(siloDir, 0755); err != nil {
		t.Fatalf("mkdir silo config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(siloDir, "home.user.nix"), []byte("{\n  config,\n  pkgs,\n  ...\n}:\n{\n}\n"), 0644); err != nil {
		t.Fatalf("write home.user.nix: %v", err)
	}
	if err := os.WriteFile(filepath.Join(siloDir, "silo.user.toml"), []byte(fmt.Sprintf("[general]\nuser = %q\n", user)), 0644); err != nil {
		t.Fatalf("write silo.user.toml: %v", err)
	}
}

// MinimalMergedConfig returns a MergedConfig suitable for use in unit tests.
func MinimalMergedConfig(id, user string) MergedConfig {
	return MergedConfig{
		ID:          id,
		User:        user,
		Features:    FeaturesConfig{Podman: false},
		Persistence: PersistenceConfig{SharedPaths: []string{}, PrivatePaths: []string{}},
		Podman:      PodmanConfig{CreateArgs: []string{}},
		Network:     NetworkConfig{Ports: []string{}},
		Limits:      LimitsConfig{},
	}
}

// MinimalWorkspaceConfig returns a WorkspaceConfig suitable for use in unit tests.
func MinimalWorkspaceConfig(id string) WorkspaceConfig {
	return WorkspaceConfig{
		General:     WorkspaceGeneralConfig{ID: id},
		Features:    FeaturesConfig{Podman: false},
		Persistence: PersistenceConfig{SharedPaths: []string{}, PrivatePaths: []string{}},
		Podman:      PodmanConfig{CreateArgs: []string{}},
		Network:     NetworkConfig{Ports: []string{}},
		Limits:      LimitsConfig{},
	}
}

// MakeDirReadonly creates a directory at path that cannot be modified.
// This is useful for testing error paths where directory creation fails.
func MakeDirReadonly(t *testing.T, path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0444); err != nil {
		t.Fatal(err)
	}
}