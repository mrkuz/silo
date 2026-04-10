package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// tomlEncode is a test helper that encodes a Config to a file using BurntSushi/toml.
func tomlEncode(f *os.File, c Config) error {
	enc := toml.NewEncoder(f)
	enc.Indent = ""
	return enc.Encode(c)
}

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
	if generateID() == generateID() {
		t.Error("two consecutive IDs should not be equal (extremely unlikely)")
	}
}

func TestBaseImageName(t *testing.T) {
	if got := baseImageName("alice"); got != "silo-alice" {
		t.Errorf("got %q, want %q", got, "silo-alice")
	}
}

func TestTOMLRoundtrip(t *testing.T) {
	original := Config{
		General: GeneralConfig{
			ID:            "abc12345",
			User:          "testuser",
			ContainerName: "silo-abc12345",
			ImageName:     "silo-abc12345",
		},
		Features: FeaturesConfig{
			SharedVolume: false,
			Nested:       true,
		},
		SharedVolume: SharedVolumeConfig{
			Paths: []string{".cache/uv/", ".local/share/opencode/"},
		},
		Connect: ConnectConfig{
			Command: "/bin/sh",
		},
		Create: CreateConfig{
			Arguments: []string{"--memory", "512m"},
		},
	}

	tmpDir := t.TempDir()
	// Temporarily override siloDir/siloToml is not possible without refactor;
	// use parseTOML directly with a temp file.
	f, err := os.CreateTemp(tmpDir, "silo-test-*.toml")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	if err := tomlEncode(f, original); err != nil {
		t.Fatal(err)
	}
	f.Close()

	parsed, err := parseTOML(path)
	if err != nil {
		t.Fatal(err)
	}

	if parsed.General != original.General {
		t.Errorf("General mismatch: got %+v, want %+v", parsed.General, original.General)
	}
	if parsed.Features != original.Features {
		t.Errorf("Features mismatch: got %+v, want %+v", parsed.Features, original.Features)
	}
	if parsed.Connect.Command != original.Connect.Command {
		t.Errorf("Command: got %q, want %q", parsed.Connect.Command, original.Connect.Command)
	}
	if len(parsed.SharedVolume.Paths) != len(original.SharedVolume.Paths) {
		t.Fatalf("SharedVolume.Paths len: got %d, want %d", len(parsed.SharedVolume.Paths), len(original.SharedVolume.Paths))
	}
	for i, want := range original.SharedVolume.Paths {
		if parsed.SharedVolume.Paths[i] != want {
			t.Errorf("SharedVolume.Paths[%d]: got %q, want %q", i, parsed.SharedVolume.Paths[i], want)
		}
	}
	if len(parsed.Create.Arguments) != len(original.Create.Arguments) {
		t.Fatalf("Create.Arguments len: got %d, want %d", len(parsed.Create.Arguments), len(original.Create.Arguments))
	}
	for i, want := range original.Create.Arguments {
		if parsed.Create.Arguments[i] != want {
			t.Errorf("Create.Arguments[%d]: got %q, want %q", i, parsed.Create.Arguments[i], want)
		}
	}
}

func TestTOMLEmptyArguments(t *testing.T) {
	cfg := Config{
		General:      GeneralConfig{ID: "x", User: "u", ContainerName: "silo-x", ImageName: "silo-x"},
		Features:     FeaturesConfig{SharedVolume: true, Nested: false},
		SharedVolume: SharedVolumeConfig{Paths: []string{}},
		Connect:      ConnectConfig{Command: "/bin/sh"},
	}

	f, err := os.CreateTemp("", "silo-test-*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	if err := tomlEncode(f, cfg); err != nil {
		t.Fatal(err)
	}
	f.Close()

	parsed, err := parseTOML(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Connect.Command != "/bin/sh" {
		t.Errorf("expected command /bin/sh, got %q", parsed.Connect.Command)
	}
	if len(parsed.Create.Arguments) != 0 {
		t.Errorf("expected empty Arguments, got %v", parsed.Create.Arguments)
	}
}

func renderSetupScript(paths []string) (string, error) {
	entries := buildSharedVolumeEntries(paths)
	got, err := renderTemplate("setup.sh.tmpl", TemplateContext{SharedPathEntries: entries})
	if err != nil {
		return "", err
	}
	return string(got), nil
}

func TestRenderSetupScript(t *testing.T) {
	t.Run("empty paths", func(t *testing.T) {
		s, err := renderSetupScript(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(s, "#!/usr/bin/env bash") {
			t.Errorf("expected shebang, got:\n%s", s)
		}
		if strings.Contains(s, "mkdir") || strings.Contains(s, "ln ") {
			t.Errorf("expected no commands for empty paths, got:\n%s", s)
		}
	})

	t.Run("directory path with $HOME prefix", func(t *testing.T) {
		s, err := renderSetupScript([]string{"$HOME/.cache/uv/"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(s, `src="/silo/shared${HOME}/.cache/uv"`) {
			t.Errorf("expected src with expanded HOME, got:\n%s", s)
		}
		if !strings.Contains(s, `dst="$HOME/.cache/uv"`) {
			t.Errorf("expected dst, got:\n%s", s)
		}
		if !strings.Contains(s, `ln -sfn "$src" "$dst"`) {
			t.Errorf("expected directory symlink, got:\n%s", s)
		}
	})

	t.Run("file path with $HOME prefix", func(t *testing.T) {
		s, err := renderSetupScript([]string{"$HOME/.local/share/fish/fish_history"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(s, `touch "$src"`) {
			t.Errorf("expected touch for file path, got:\n%s", s)
		}
		if !strings.Contains(s, `ln -sf "$src" "$dst"`) {
			t.Errorf("expected file symlink, got:\n%s", s)
		}
	})

	t.Run("absolute directory path", func(t *testing.T) {
		s, err := renderSetupScript([]string{"/etc/foo/"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(s, `src="/silo/shared/etc/foo"`) {
			t.Errorf("expected absolute src, got:\n%s", s)
		}
		if !strings.Contains(s, `dst="/etc/foo"`) {
			t.Errorf("expected absolute dst, got:\n%s", s)
		}
	})

	t.Run("absolute file path", func(t *testing.T) {
		s, err := renderSetupScript([]string{"/home/alice/.gitconfig"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(s, `src="/silo/shared/home/alice/.gitconfig"`) {
			t.Errorf("expected absolute src, got:\n%s", s)
		}
		if !strings.Contains(s, `touch "$src"`) {
			t.Errorf("expected touch for file, got:\n%s", s)
		}
	})

	t.Run("multiple paths", func(t *testing.T) {
		s, err := renderSetupScript([]string{"$HOME/.cache/uv/", "$HOME/.cache/pip/"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(s, ".cache/uv") || !strings.Contains(s, ".cache/pip") {
			t.Errorf("expected both paths in script, got:\n%s", s)
		}
	})

	t.Run("guards against existing non-symlink", func(t *testing.T) {
		s, err := renderSetupScript([]string{"$HOME/.cache/uv/"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(s, `[ -L "$dst" ] || [ ! -e "$dst" ]`) {
			t.Errorf("expected guard check before symlink, got:\n%s", s)
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg, err := defaultConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.General.ID) != 8 {
		t.Errorf("expected ID length 8, got %d", len(cfg.General.ID))
	}
	if cfg.General.ContainerName != "silo-"+cfg.General.ID {
		t.Errorf("ContainerName %q does not match ID %q", cfg.General.ContainerName, cfg.General.ID)
	}
	if cfg.General.ImageName != "silo-"+cfg.General.ID {
		t.Errorf("ImageName %q does not match ID %q", cfg.General.ImageName, cfg.General.ID)
	}
	if cfg.General.User == "" {
		t.Error("expected non-empty User")
	}
	if cfg.Connect.Command != "/bin/sh" {
		t.Errorf("expected command /bin/sh, got %q", cfg.Connect.Command)
	}
	if cfg.Features.SharedVolume || cfg.Features.Nested {
		t.Errorf("unexpected feature defaults: %+v", cfg.Features)
	}
	if cfg.SharedVolume.Paths == nil {
		t.Error("expected non-nil SharedVolume.Paths")
	}
	if cfg.Create.Arguments == nil {
		t.Error("expected non-nil Create.Arguments")
	}
}

func TestLoadSiloInTOML(t *testing.T) {
	t.Run("returns empty config when file absent", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		cfg, err := loadSiloInTOML()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Connect.Command != "" || cfg.General.ID != "" {
			t.Errorf("expected zero config for absent file, got %+v", cfg)
		}
	})

	t.Run("parses connect and features from existing file", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		siloConfigPath := filepath.Join(base, "silo", "silo.in.toml")
		if err := os.MkdirAll(filepath.Dir(siloConfigPath), 0755); err != nil {
			t.Fatal(err)
		}
		content := []byte("[connect]\ncommand = \"/bin/fish\"\n[features]\nnested = true\n")
		if err := os.WriteFile(siloConfigPath, content, 0644); err != nil {
			t.Fatal(err)
		}
		cfg, err := loadSiloInTOML()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Connect.Command != "/bin/fish" {
			t.Errorf("expected command /bin/fish, got %q", cfg.Connect.Command)
		}
		if !cfg.Features.Nested {
			t.Error("expected Features.Nested = true")
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
		_, err := loadSiloInTOML()
		if err == nil {
			t.Error("expected error for malformed TOML")
		}
	})
}

func TestUserConfigDir(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME when set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
		got, err := userConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "/tmp/xdg-test/silo" {
			t.Errorf("got %q, want %q", got, "/tmp/xdg-test/silo")
		}
	})

	t.Run("falls back to HOME/.config/silo", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		got, err := userConfigDir()
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
		if err := ensureFile(path, []byte("hello")); err != nil {
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
		if err := ensureFile(path, []byte("new")); err != nil {
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
		setupUserConfig(t)
		setupWorkspace(t, Config{}) // write an *empty* silo.toml so setupWorkspace doesn't interfere
		// Remove the file so we simulate a true first run.
		os.Remove(siloToml)
		os.Remove(siloDir) // remove dir too so it is recreated

		cfg, firstRun, err := initWorkspaceConfig()
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

	t.Run("first run: seeds Connect.Command from user config", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		siloUser := filepath.Join(base, "silo")
		if err := os.MkdirAll(siloUser, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(siloUser, "home-user.nix"), []byte(emptyHomeNix), 0644); err != nil {
			t.Fatal(err)
		}
		userToml := filepath.Join(siloUser, "silo.in.toml")
		if err := os.WriteFile(userToml, []byte("[connect]\ncommand = \"/bin/fish\"\n"), 0644); err != nil {
			t.Fatal(err)
		}

		// Workspace directory with no silo.toml.
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)

		cfg, firstRun, err := initWorkspaceConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !firstRun {
			t.Error("expected firstRun=true on first run")
		}
		if cfg.Connect.Command != "/bin/fish" {
			t.Errorf("expected seeded command /bin/fish, got %q", cfg.Connect.Command)
		}
	})

	t.Run("second run: existing silo.toml is loaded unchanged", func(t *testing.T) {
		existing := minimalConfig("deadbeef")
		existing.Connect.Command = "/usr/bin/zsh"
		setupWorkspace(t, existing)
		setupUserConfig(t)

		cfg, firstRun, err := initWorkspaceConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if firstRun {
			t.Error("expected firstRun=false on subsequent run")
		}
		if cfg.General.ID != "deadbeef" {
			t.Errorf("expected ID deadbeef, got %q", cfg.General.ID)
		}
		if cfg.Connect.Command != "/usr/bin/zsh" {
			t.Errorf("expected command /usr/bin/zsh, got %q", cfg.Connect.Command)
		}
	})
}

func TestEnsureUserFiles(t *testing.T) {
	t.Run("creates user files when absent", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		if err := ensureUserFiles(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		dir := filepath.Join(base, "silo")
		for _, name := range []string{"home-user.nix", "devcontainer.in.json", "silo.in.toml"} {
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
		os.WriteFile(filepath.Join(dir, "home-user.nix"), sentinel, 0644)

		if err := ensureUserFiles(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, _ := os.ReadFile(filepath.Join(dir, "home-user.nix"))
		if string(got) != string(sentinel) {
			t.Errorf("home-user.nix was overwritten")
		}
	})
}

func TestEnsureUserFilesError(t *testing.T) {
	t.Run("returns error when user config directory cannot be created", func(t *testing.T) {
		// Set XDG_CONFIG_HOME to a path in a non-existent directory to force failure
		t.Setenv("XDG_CONFIG_HOME", "/nonexistent/deeply/nested/path")
		err := ensureUserFiles()
		if err == nil {
			t.Error("expected error when user config directory is inaccessible")
		}
	})
}

func TestEnsureInitError(t *testing.T) {
	t.Run("returns error when workspace files cannot be created", func(t *testing.T) {
		// Create a read-only directory to force ensureWorkspaceFiles to fail
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)

		// Create .silo as a file (not directory) to cause mkdirall to fail
		if err := os.WriteFile(siloDir, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		// Make it read-only
		if err := os.Chmod(siloDir, 0444); err != nil {
			t.Fatal(err)
		}

		setupUserConfig(t)
		_, _, err := ensureInit()
		if err == nil {
			t.Error("expected error when workspace files cannot be created")
		}
	})
}

func TestEnsureWorkspaceFiles(t *testing.T) {
	t.Run("creates workspace home.nix when absent", func(t *testing.T) {
		setupWorkspace(t, minimalConfig("abc12345"))
		os.Remove(filepath.Join(siloDir, "home.nix"))
		if err := ensureWorkspaceFiles(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(filepath.Join(siloDir, "home.nix")); os.IsNotExist(err) {
			t.Error("expected .silo/home.nix to be created")
		}
	})

	t.Run("does not overwrite existing .silo/home.nix", func(t *testing.T) {
		setupWorkspace(t, minimalConfig("abc12345"))
		sentinel := []byte("# custom\n")
		os.WriteFile(filepath.Join(siloDir, "home.nix"), sentinel, 0644)
		if err := ensureWorkspaceFiles(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, _ := os.ReadFile(filepath.Join(siloDir, "home.nix"))
		if string(got) != string(sentinel) {
			t.Errorf("workspace home.nix was overwritten")
		}
	})
}

func TestSaveWorkspaceConfigTOMLFormat(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	cfg := minimalConfig("abc12345")
	cfg.Connect.Command = "fish --login"
	cfg.Features.SharedVolume = true
	cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/", "$HOME/.local/share/opencode/"}
	if err := cfg.saveWorkspaceConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, err := os.ReadFile(siloToml)
	if err != nil {
		t.Fatalf("read silo.toml: %v", err)
	}
	assertTOMLFormat(t, string(raw))
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
			if i > 0 && lines[i-1] != "" {
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

func TestSaveWorkspaceConfigNilGuards(t *testing.T) {
	// Configs loaded from old TOML files may have nil slices;
	// saveWorkspaceConfig must normalize them to empty slices.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	cfg := minimalConfig("abc12345")
	cfg.SharedVolume.Paths = nil
	cfg.Create.Arguments = nil
	if err := cfg.saveWorkspaceConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parsed, err := parseTOML(siloToml)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if parsed.SharedVolume.Paths == nil {
		t.Error("SharedVolume.Paths should not be nil after save")
	}
	if parsed.Create.Arguments == nil {
		t.Error("Create.Arguments should not be nil after save")
	}
}
