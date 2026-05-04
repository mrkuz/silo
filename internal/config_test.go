package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestGenerateID(t *testing.T) {
	id := generateID()
	if len(id) != 8 {
		t.Errorf("expected length 8, got %d", len(id))
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			t.Errorf("unexpected character %q in ID", c)
		}
	}
	// IDs should be unique
	id1, id2 := generateID(), generateID()
	if id1 == id2 {
		t.Error("two consecutive IDs should not be equal (extremely unlikely)")
	}
}

func TestBaseImageName(t *testing.T) {
	if got := BaseImageName("alice"); got != "silo-alice" {
		t.Errorf("got %q, want %q", got, "silo-alice")
	}
}

func TestTOMLRoundtrip(t *testing.T) {
	original := Config{
		General: GeneralConfig{
			ID:   "abc12345",
			User: "testuser",
		},
		Features: FeaturesConfig{
			Podman:       true,
		},
		SharedVolume: SharedVolumeConfig{
			Paths: []string{".cache/uv/", ".local/share/opencode/"},
		},
		Podman: PodmanConfig{
			CreateArgs: []string{"--memory", "512m"},
		},
	}

	tmpDir := t.TempDir()
	f, err := os.CreateTemp(tmpDir, "silo-test-*.toml")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	if err := toml.NewEncoder(f).Encode(original); err != nil {
		t.Fatal(err)
	}
	f.Close()

	parsed, err := ParseTOML(path)
	if err != nil {
		t.Fatal(err)
	}

	if parsed.General != original.General {
		t.Errorf("General mismatch: got %+v, want %+v", parsed.General, original.General)
	}
	if parsed.Features != original.Features {
		t.Errorf("Features mismatch: got %+v, want %+v", parsed.Features, original.Features)
	}
	if len(parsed.SharedVolume.Paths) != len(original.SharedVolume.Paths) {
		t.Fatalf("SharedVolume.Paths len: got %d, want %d", len(parsed.SharedVolume.Paths), len(original.SharedVolume.Paths))
	}
	for i, want := range original.SharedVolume.Paths {
		if parsed.SharedVolume.Paths[i] != want {
			t.Errorf("SharedVolume.Paths[%d]: got %q, want %q", i, parsed.SharedVolume.Paths[i], want)
		}
	}
	if len(parsed.Podman.CreateArgs) != len(original.Podman.CreateArgs) {
		t.Fatalf("CreateArgs len: got %d, want %d", len(parsed.Podman.CreateArgs), len(original.Podman.CreateArgs))
	}
	for i, want := range original.Podman.CreateArgs {
		if parsed.Podman.CreateArgs[i] != want {
			t.Errorf("CreateArgs[%d]: got %q, want %q", i, parsed.Podman.CreateArgs[i], want)
		}
	}
}

func TestTOMLEmptyCreateArgs(t *testing.T) {
	cfg := Config{
		General:      GeneralConfig{ID: "x", User: "u"},
		Features:     FeaturesConfig{Podman: false},
		SharedVolume: SharedVolumeConfig{Name: "silo-shared", Paths: []string{}},
	}

	f, err := os.CreateTemp("", "silo-test-*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		t.Fatal(err)
	}
	f.Close()

	parsed, err := ParseTOML(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Podman.CreateArgs) != 0 {
		t.Errorf("expected empty CreateArgs, got %v", parsed.Podman.CreateArgs)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg, err := DefaultConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.General.ID) != 8 {
		t.Errorf("expected ID length 8, got %d", len(cfg.General.ID))
	}
	if WorkspaceContainerName(cfg.General.ID) != "silo-"+cfg.General.ID {
		t.Errorf("WorkspaceContainerName %q does not match expected %q", WorkspaceContainerName(cfg.General.ID), "silo-"+cfg.General.ID)
	}
	if WorkspaceImageName(cfg.General.ID) != "silo-"+cfg.General.ID {
		t.Errorf("WorkspaceImageName %q does not match expected %q", WorkspaceImageName(cfg.General.ID), "silo-"+cfg.General.ID)
	}
	if cfg.General.User == "" {
		t.Error("expected non-empty User")
	}
	if cfg.Features.Podman {
		t.Errorf("unexpected feature defaults: %+v", cfg.Features)
	}
	if cfg.SharedVolume.Paths == nil {
		t.Error("expected non-nil SharedVolume.Paths")
	}
	if cfg.Podman.CreateArgs == nil {
		t.Error("expected non-nil CreateArgs")
	}
}

func TestLoadSiloInTOML(t *testing.T) {
	t.Run("returns empty config when file absent", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		cfg, err := LoadSiloInTOML()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.General.ID != "" {
			t.Errorf("expected zero config for absent file, got %+v", cfg)
		}
	})

	t.Run("parses features from existing file", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		siloConfigPath := filepath.Join(base, "silo", "silo.in.toml")
		if err := os.MkdirAll(filepath.Dir(siloConfigPath), 0755); err != nil {
			t.Fatal(err)
		}
		content := []byte("[features]\npodman = true\n")
		if err := os.WriteFile(siloConfigPath, content, 0644); err != nil {
			t.Fatal(err)
		}
		cfg, err := LoadSiloInTOML()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cfg.Features.Podman {
			t.Error("expected Features.Podman = true")
		}
	})

	t.Run("malformed TOML returns error", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		siloConfigPath := filepath.Join(base, "silo", "silo.in.toml")
		if err := os.MkdirAll(filepath.Dir(siloConfigPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(siloConfigPath, []byte("invalid = [toml"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := LoadSiloInTOML()
		if err == nil {
			t.Error("expected error for malformed TOML")
		}
	})
}

func TestUserConfigDir(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME when set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
		got, err := UserConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "/tmp/xdg-test/silo" {
			t.Errorf("got %q, want %q", got, "/tmp/xdg-test/silo")
		}
	})

	t.Run("falls back to HOME/.config/silo", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		got, err := UserConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasSuffix(got, "/.config/silo") {
			t.Errorf("expected path ending in /.config/silo, got %q", got)
		}
	})
}

func TestEnsureFile(t *testing.T) {
	t.Run("creates file when absent", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/sub/file.txt"
		if err := EnsureFile(path, []byte("hello")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("file not created: %v", err)
		}
		if string(got) != "hello" {
			t.Errorf("got %q, want %q", string(got), "hello")
		}
	})

	t.Run("does not overwrite existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/file.txt"
		if err := os.WriteFile(path, []byte("original"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := EnsureFile(path, []byte("new")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "original" {
			t.Errorf("got %q, want file unchanged %q", string(got), "original")
		}
	})
}

func TestInitWorkspaceConfig(t *testing.T) {
	t.Run("first run: creates .silo/silo.toml with generated ID", func(t *testing.T) {
		SetupUserConfig(t)
		SetupWorkspace(t, Config{}) // write an *empty* silo.toml so setupWorkspace doesn't interfere
		// Remove the file so we simulate a true first run.
		os.Remove(SiloToml())
		os.Remove(SiloDir()) // remove dir too so it is recreated

		cfg, firstRun, err := InitWorkspaceConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !firstRun {
			t.Error("expected firstRun=true on first run")
		}
		if len(cfg.General.ID) != 8 {
			t.Errorf("expected 8-char ID, got %q", cfg.General.ID)
		}
	})

	t.Run("second run: existing silo.toml is loaded unchanged", func(t *testing.T) {
		existing := MinimalConfig("deadbeef")
		SetupWorkspace(t, existing)
		SetupUserConfig(t)

		cfg, firstRun, err := InitWorkspaceConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if firstRun {
			t.Error("expected firstRun=false on subsequent run")
		}
		if cfg.General.ID != "deadbeef" {
			t.Errorf("expected ID deadbeef, got %q", cfg.General.ID)
		}
	})
}

func TestEnsureUserFiles(t *testing.T) {
	t.Run("creates user files when absent", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		if err := EnsureUserFiles(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		dir := filepath.Join(base, "silo")
		for _, name := range []string{"home.user.nix", "devcontainer.in.json", "silo.in.toml"} {
			if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
				t.Errorf("expected %s to be created", name)
			}
		}
	})

	t.Run("does not overwrite existing files", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		dir := filepath.Join(base, "silo")
		os.MkdirAll(dir, 0755)
		sentinel := []byte("# custom\n")
		os.WriteFile(filepath.Join(dir, "home.user.nix"), sentinel, 0644)

		if err := EnsureUserFiles(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, _ := os.ReadFile(filepath.Join(dir, "home.user.nix"))
		if string(got) != string(sentinel) {
			t.Errorf("home.user.nix was overwritten")
		}
	})
}

func TestEnsureUserFilesError(t *testing.T) {
	t.Run("returns error when user config directory cannot be created", func(t *testing.T) {
		// Set XDG_CONFIG_HOME to a path in a non-existent directory to force failure
		t.Setenv("XDG_CONFIG_HOME", "/nonexistent/deeply/nested/path")
		err := EnsureUserFiles()
		if err == nil {
			t.Error("expected error when user config directory is inaccessible")
		}
	})
}

func TestEnsureInitError(t *testing.T) {
	t.Run("returns error when workspace files cannot be created", func(t *testing.T) {
		// Create a read-only directory to force EnsureWorkspaceFiles to fail
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)

		// Create .silo as a file (not directory) to cause mkdirall to fail
		if err := os.WriteFile(SiloDir(), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		// Make it read-only
		if err := os.Chmod(SiloDir(), 0444); err != nil {
			t.Fatal(err)
		}

		SetupUserConfig(t)
		_, _, err := EnsureInit(nil)
		if err == nil {
			t.Error("expected error when workspace files cannot be created")
		}
	})
}

func TestEnsureWorkspaceFiles(t *testing.T) {
	t.Run("creates workspace home.nix when absent", func(t *testing.T) {
		SetupWorkspace(t, MinimalConfig("abc12345"))
		os.Remove(filepath.Join(SiloDir(), "home.nix"))
		if err := EnsureWorkspaceFiles(false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(filepath.Join(SiloDir(), "home.nix")); os.IsNotExist(err) {
			t.Error("expected .silo/home.nix to be created")
		}
	})

	t.Run("does not overwrite existing .silo/home.nix", func(t *testing.T) {
		SetupWorkspace(t, MinimalConfig("abc12345"))
		sentinel := []byte("# custom\n")
		os.WriteFile(filepath.Join(SiloDir(), "home.nix"), sentinel, 0644)
		if err := EnsureWorkspaceFiles(false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, _ := os.ReadFile(filepath.Join(SiloDir(), "home.nix"))
		if string(got) != string(sentinel) {
			t.Errorf("workspace home.nix was overwritten")
		}
	})

	t.Run("podman=true creates workspace home.nix with podman enabled", func(t *testing.T) {
		SetupWorkspace(t, MinimalConfig("abc12345"))
		os.Remove(filepath.Join(SiloDir(), "home.nix"))
		if err := EnsureWorkspaceFiles(true); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(SiloDir(), "home.nix"))
		if err != nil {
			t.Fatalf("failed to read workspace home.nix: %v", err)
		}
		if !strings.Contains(string(content), "silo.podman.enable = true") {
			t.Errorf("workspace home.nix should contain 'silo.podman.enable = true', got: %s", content)
		}
	})

	t.Run("podman=false creates workspace home.nix with podman disabled", func(t *testing.T) {
		SetupWorkspace(t, MinimalConfig("abc12345"))
		os.Remove(filepath.Join(SiloDir(), "home.nix"))
		if err := EnsureWorkspaceFiles(false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(SiloDir(), "home.nix"))
		if err != nil {
			t.Fatalf("failed to read workspace home.nix: %v", err)
		}
		if !strings.Contains(string(content), "silo.podman.enable = false") {
			t.Errorf("workspace home.nix should contain 'silo.podman.enable = false', got: %s", content)
		}
	})
}

func TestSaveWorkspaceConfigTOMLFormat(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	cfg := MinimalConfig("abc12345")
	cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/", "$HOME/.local/share/opencode/"}
	if err := cfg.SaveWorkspaceConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, err := os.ReadFile(SiloToml())
	if err != nil {
		t.Fatalf("read silo.toml: %v", err)
	}
	assertTOMLFormat(t, string(raw))
}

func TestSaveWorkspaceConfigNilGuards(t *testing.T) {
	// Configs loaded from old TOML files may have nil slices;
	// SaveWorkspaceConfig must normalize them to empty slices.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	cfg := MinimalConfig("abc12345")
	cfg.SharedVolume.Paths = nil
	cfg.Podman.CreateArgs = nil
	if err := cfg.SaveWorkspaceConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parsed, err := ParseTOML(SiloToml())
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if parsed.SharedVolume.Paths == nil {
		t.Error("SharedVolume.Paths should not be nil after save")
	}
	if parsed.Podman.CreateArgs == nil {
		t.Error("CreateArgs should not be nil after save")
	}
}

// assertTOMLFormat checks that s follows silo's TOML style:
//   - no tab characters anywhere
//   - keys are not indented; only array string elements use exactly 2-space indent
//   - a blank line precedes each [section] header
func assertTOMLFormat(t *testing.T, s string) {
	t.Helper()
	if strings.ContainsRune(s, '\t') {
		t.Error("TOML must not contain tab characters")
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		switch {
		case line == "" || line == "]":
			// blank lines and closing brackets are fine
		case strings.HasPrefix(line, "["):
			// section header: must be preceded by a blank line (except first)
			// exception: direct tables like [podman] don't need a blank line after [podman]
			if i > 0 && lines[i-1] != "" && lines[i-1] != "]" && !strings.HasPrefix(line, "[podman]") {
				t.Errorf("line %d: expected blank line before section header, got %q", i+1, lines[i-1])
			}
		case strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t"):
			// indented line: must be a quoted array element with exactly 2 spaces
			if !strings.HasPrefix(strings.TrimSpace(line), `"`) {
				t.Errorf("line %d: unexpected indent on non-array line: %q", i+1, line)
			} else if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "   ") {
				t.Errorf("line %d: array element must use exactly 2-space indent, got %q", i+1, line)
			}
		}
	}
}