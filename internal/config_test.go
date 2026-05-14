package internal

import (
	"os"
	"path/filepath"
	"slices"
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

func TestTOMLRoundtrip(t *testing.T) {
	original := WorkspaceConfig{
		General: WorkspaceGeneralConfig{
			ID: "abc12345",
		},
		Features: FeaturesConfig{
			Podman: true,
		},
		Persistence: PersistenceConfig{
			SharedPaths: []string{".cache/uv/", ".local/share/opencode/"},
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

	var parsed WorkspaceConfig
	if err := ParseTOML(path, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed.General != original.General {
		t.Errorf("General mismatch: got %+v, want %+v", parsed.General, original.General)
	}
	if parsed.Features != original.Features {
		t.Errorf("Features mismatch: got %+v, want %+v", parsed.Features, original.Features)
	}
	if len(parsed.Persistence.SharedPaths) != len(original.Persistence.SharedPaths) {
		t.Fatalf("Persistence.SharedPaths len: got %d, want %d", len(parsed.Persistence.SharedPaths), len(original.Persistence.SharedPaths))
	}
	for i, want := range original.Persistence.SharedPaths {
		if parsed.Persistence.SharedPaths[i] != want {
			t.Errorf("Persistence.SharedPaths[%d]: got %q, want %q", i, parsed.Persistence.SharedPaths[i], want)
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
	cfg := WorkspaceConfig{
		General:     WorkspaceGeneralConfig{ID: "x"},
		Features:    FeaturesConfig{Podman: false},
		Persistence: PersistenceConfig{SharedPaths: []string{}},
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

	var parsed WorkspaceConfig
	if err := ParseTOML(f.Name(), &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Podman.CreateArgs) != 0 {
		t.Errorf("expected empty CreateArgs, got %v", parsed.Podman.CreateArgs)
	}
}

func TestDefaultWorkspaceConfig(t *testing.T) {
	cfg, err := DefaultWorkspaceConfig()
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
	if cfg.Features.Podman {
		t.Errorf("unexpected feature defaults: %+v", cfg.Features)
	}
	if cfg.Persistence.SharedPaths == nil {
		t.Error("expected non-nil Persistence.SharedPaths")
	}
	if cfg.Podman.CreateArgs == nil {
		t.Error("expected non-nil CreateArgs")
	}
	if cfg.Limits.CPUs != 0 {
		t.Errorf("expected default CPUs=0, got %d", cfg.Limits.CPUs)
	}
	if cfg.Limits.Memory != 0 {
		t.Errorf("expected default Memory=0, got %d", cfg.Limits.Memory)
	}
	if cfg.Limits.Processes != 1024 {
		t.Errorf("expected default Processes=1024, got %d", cfg.Limits.Processes)
	}
}

func TestLoadSiloUserTOML(t *testing.T) {
	t.Run("returns error when file absent", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		_, err := LoadSiloUserTOML()
		if err == nil {
			t.Error("expected error when file absent")
		}
	})

	t.Run("parses user from existing file", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		siloConfigPath := filepath.Join(base, "silo", "silo.user.toml")
		if err := os.MkdirAll(filepath.Dir(siloConfigPath), 0755); err != nil {
			t.Fatal(err)
		}
		content := []byte("[general]\nuser = \"alice\"\n")
		if err := os.WriteFile(siloConfigPath, content, 0644); err != nil {
			t.Fatal(err)
		}
		cfg, err := LoadSiloUserTOML()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.General.User != "alice" {
			t.Errorf("expected User alice, got %q", cfg.General.User)
		}
	})

	t.Run("missing general.user returns error", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		siloConfigPath := filepath.Join(base, "silo", "silo.user.toml")
		if err := os.MkdirAll(filepath.Dir(siloConfigPath), 0755); err != nil {
			t.Fatal(err)
		}
		content := []byte("[features]\npodman = true\n")
		if err := os.WriteFile(siloConfigPath, content, 0644); err != nil {
			t.Fatal(err)
		}
		_, err := LoadSiloUserTOML()
		if err == nil {
			t.Error("expected error when [general].user is missing")
		}
	})

	t.Run("malformed TOML returns error", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		siloConfigPath := filepath.Join(base, "silo", "silo.user.toml")
		if err := os.MkdirAll(filepath.Dir(siloConfigPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(siloConfigPath, []byte("invalid = [toml"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := LoadSiloUserTOML()
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

func TestEnsureInit(t *testing.T) {
	t.Run("first run: creates .silo/silo.toml with generated ID", func(t *testing.T) {
		SetupUserFiles(t)
		SetupEmptyWorkspace(t)

		cfg, err := EnsureInit(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.General.ID) != 8 {
			t.Errorf("expected 8-char ID, got %q", cfg.General.ID)
		}
	})

	t.Run("second run: existing silo.toml is loaded unchanged", func(t *testing.T) {
		existing := MinimalWorkspaceConfig("deadbeef")
		SetupWorkspaceFiles(t, existing)
		SetupUserFiles(t)

		cfg, err := EnsureInit(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.General.ID != "deadbeef" {
			t.Errorf("expected ID deadbeef, got %q", cfg.General.ID)
		}
	})
}

func TestRequireMergedConfig(t *testing.T) {
	t.Run("missing ID returns error", func(t *testing.T) {
		SetupUserFiles(t)
		cfg := MinimalWorkspaceConfig("abc12345")
		cfg.General.ID = ""
		SetupWorkspaceFiles(t, cfg)

		_, err := RequireMergedConfig()
		if err == nil {
			t.Error("expected error when [general].id is empty")
		}
		if !strings.Contains(err.Error(), "[general].id is required") {
			t.Errorf("expected error to mention '[general].id is required', got: %v", err)
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
		for _, name := range []string{"home.user.nix", "devcontainer.user.json", "silo.user.toml"} {
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
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)

		MakeDirReadonly(t, SiloDir())

		SetupUserFiles(t)
		_, err := EnsureInit(nil)
		if err == nil {
			t.Error("expected error when workspace files cannot be created")
		}
	})
}

func TestEnsureWorkspaceFiles(t *testing.T) {
	t.Run("creates workspace home.nix when absent", func(t *testing.T) {
		SetupWorkspaceFiles(t, MinimalWorkspaceConfig("abc12345"))
		os.Remove(filepath.Join(SiloDir(), "home.nix"))
		if err := EnsureWorkspaceFiles(false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(filepath.Join(SiloDir(), "home.nix")); os.IsNotExist(err) {
			t.Error("expected .silo/home.nix to be created")
		}
	})

	t.Run("does not overwrite existing .silo/home.nix", func(t *testing.T) {
		SetupWorkspaceFiles(t, MinimalWorkspaceConfig("abc12345"))
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
		SetupWorkspaceFiles(t, MinimalWorkspaceConfig("abc12345"))
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
		SetupWorkspaceFiles(t, MinimalWorkspaceConfig("abc12345"))
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
	SetupWorkspaceFiles(t, MinimalWorkspaceConfig("abc12345"))

	cfg := MinimalWorkspaceConfig("abc12345")
	cfg.Persistence.SharedPaths = []string{"$HOME/.cache/uv/", "$HOME/.local/share/opencode/"}
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
	SetupWorkspaceFiles(t, MinimalWorkspaceConfig("abc12345"))

	cfg := MinimalWorkspaceConfig("abc12345")
	cfg.Persistence.SharedPaths = nil
	cfg.Podman.CreateArgs = nil
	if err := cfg.SaveWorkspaceConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed WorkspaceConfig
	if err := ParseTOML(SiloToml(), &parsed); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if parsed.Persistence.SharedPaths == nil {
		t.Error("Persistence.SharedPaths should not be nil after save")
	}
	if parsed.Podman.CreateArgs == nil {
		t.Error("CreateArgs should not be nil after save")
	}
}

func TestWorkspaceNames(t *testing.T) {
	id := "abc12345"
	if got := WorkspaceContainerName(id); got != "silo-abc12345" {
		t.Errorf("WorkspaceContainerName(%q) = %q, want %q", id, got, "silo-abc12345")
	}
	if got := WorkspaceImageName(id); got != "silo-abc12345" {
		t.Errorf("WorkspaceImageName(%q) = %q, want %q", id, got, "silo-abc12345")
	}
}

func TestMergeUserInto(t *testing.T) {
	t.Run("user values prepend to workspace values", func(t *testing.T) {
		workspace := WorkspaceConfig{
			General:     WorkspaceGeneralConfig{ID: "abc12345"},
			Podman:      PodmanConfig{CreateArgs: []string{"--workspace-flag"}},
			Persistence: PersistenceConfig{SharedPaths: []string{"/workspace/path"}, PrivatePaths: []string{"/workspace/private"}},
			Network:     NetworkConfig{Ports: []string{"8080:8080"}},
		}
		user := UserConfig{
			General:     UserGeneralConfig{User: "alice"},
			Podman:      UserPodmanConfig{CreateArgs: []string{"--user-flag"}, Network: NetworkConfig{Ports: []string{"3000:3000"}}},
			Persistence: PersistenceConfig{SharedPaths: []string{"$HOME/.cache"}, PrivatePaths: []string{"$HOME/.private"}},
		}
		got := MergeUserInto(workspace, user)

		if got.User != "alice" {
			t.Errorf("expected User alice, got %q", got.User)
		}
		if got.ID != "abc12345" {
			t.Errorf("expected ID abc12345, got %q", got.ID)
		}
		if !slices.Equal(got.Podman.CreateArgs, []string{"--user-flag", "--workspace-flag"}) {
			t.Errorf("expected CreateArgs [user-flag workspace-flag], got %v", got.Podman.CreateArgs)
		}
		if !slices.Equal(got.Persistence.SharedPaths, []string{"$HOME/.cache", "/workspace/path"}) {
			t.Errorf("expected SharedPaths [$HOME/.cache /workspace/path], got %v", got.Persistence.SharedPaths)
		}
		if !slices.Equal(got.Persistence.PrivatePaths, []string{"$HOME/.private", "/workspace/private"}) {
			t.Errorf("expected PrivatePaths [$HOME/.private /workspace/private], got %v", got.Persistence.PrivatePaths)
		}
		if !slices.Equal(got.Network.Ports, []string{"3000:3000", "8080:8080"}) {
			t.Errorf("expected Ports [3000:3000 8080:8080], got %v", got.Network.Ports)
		}
	})

	t.Run("features.podman from workspace only", func(t *testing.T) {
		workspace := WorkspaceConfig{
			General:  WorkspaceGeneralConfig{ID: "abc12345"},
			Features: FeaturesConfig{Podman: true},
		}
		user := UserConfig{
			General: UserGeneralConfig{User: "alice"},
		}
		got := MergeUserInto(workspace, user)
		if got.Features.Podman != true {
			t.Error("expected Features.Podman from workspace")
		}
	})

	t.Run("empty user values leave workspace intact", func(t *testing.T) {
		workspace := WorkspaceConfig{
			General:     WorkspaceGeneralConfig{ID: "abc12345"},
			Podman:      PodmanConfig{CreateArgs: []string{"--ws-flag"}},
			Persistence: PersistenceConfig{SharedPaths: []string{"/ws/path"}},
		}
		user := UserConfig{
			General:     UserGeneralConfig{User: "alice"},
			Podman:      UserPodmanConfig{CreateArgs: []string{}},
			Persistence: PersistenceConfig{SharedPaths: []string{}},
		}
		got := MergeUserInto(workspace, user)
		if !slices.Equal(got.Podman.CreateArgs, []string{"--ws-flag"}) {
			t.Errorf("expected CreateArgs [ws-flag], got %v", got.Podman.CreateArgs)
		}
		if !slices.Equal(got.Persistence.SharedPaths, []string{"/ws/path"}) {
			t.Errorf("expected SharedPaths [/ws/path], got %v", got.Persistence.SharedPaths)
		}
	})
}

func TestUserFiles(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	files, err := UserFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d", len(files))
	}
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}
	dir := filepath.Join(base, "silo")
	for _, name := range []string{"home.user.nix", "devcontainer.user.json", "silo.user.toml"} {
		found := false
		for _, p := range paths {
			if strings.HasSuffix(p, name) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %s in user files", name)
		}
		_ = dir
	}
	if !strings.HasSuffix(files[0].Path, "home.user.nix") {
		t.Errorf("expected first file to be home.user.nix, got %s", files[0].Path)
	}
	if !strings.HasSuffix(files[1].Path, "devcontainer.user.json") {
		t.Errorf("expected second file to be devcontainer.user.json, got %s", files[1].Path)
	}
	if !strings.HasSuffix(files[2].Path, "silo.user.toml") {
		t.Errorf("expected third file to be silo.user.toml, got %s", files[2].Path)
	}
}

func TestEnsureDevcontainerUserJSON(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	if err := EnsureDevcontainerUserJSON(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := filepath.Join(base, "silo", "devcontainer.user.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected devcontainer.user.json to be created")
	}
	content, _ := os.ReadFile(path)
	if string(content) != "{}\n" {
		t.Errorf("expected %q, got %q", "{}\n", string(content))
	}
}

func TestLoadSiloDevcontainerJSON(t *testing.T) {
	t.Run("returns empty map when file absent", func(t *testing.T) {
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)
		got, err := LoadSiloDevcontainerJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})

	t.Run("parses existing file", func(t *testing.T) {
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)
		if err := os.MkdirAll(".silo", 0755); err != nil {
			t.Fatal(err)
		}
		content := []byte(`{"name": "test", "custom": "value"}`)
		if err := os.WriteFile(".silo/devcontainer.json", content, 0644); err != nil {
			t.Fatal(err)
		}
		got, err := LoadSiloDevcontainerJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["name"] != "test" {
			t.Errorf("expected name=test, got %v", got["name"])
		}
		if got["custom"] != "value" {
			t.Errorf("expected custom=value, got %v", got["custom"])
		}
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)
		if err := os.MkdirAll(".silo", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(".silo/devcontainer.json", []byte("{invalid"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := LoadSiloDevcontainerJSON()
		if err == nil {
			t.Error("expected error for malformed JSON")
		}
	})
}

func TestLoadDevcontainerUserJSON(t *testing.T) {
	t.Run("returns empty map when file absent", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		got, err := LoadDevcontainerUserJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})

	t.Run("parses existing file", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		dir := filepath.Join(base, "silo")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		content := []byte(`{"extensions": ["ms-python.python"]}`)
		if err := os.WriteFile(filepath.Join(dir, "devcontainer.user.json"), content, 0644); err != nil {
			t.Fatal(err)
		}
		got, err := LoadDevcontainerUserJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		exts, ok := got["extensions"].([]any)
		if !ok {
			t.Fatalf("expected extensions array, got %v", got["extensions"])
		}
		if len(exts) != 1 || exts[0] != "ms-python.python" {
			t.Errorf("expected extensions [ms-python.python], got %v", exts)
		}
	})
}

func TestPrintRunningStatus(t *testing.T) {
	t.Run("running", func(t *testing.T) {
		output := CaptureStdout(func() { PrintRunningStatus(true) })
		if !strings.Contains(output, "Running") {
			t.Errorf("expected 'Running', got %s", output)
		}
	})

	t.Run("stopped", func(t *testing.T) {
		output := CaptureStdout(func() { PrintRunningStatus(false) })
		if !strings.Contains(output, "Stopped") {
			t.Errorf("expected 'Stopped', got %s", output)
		}
	})
}

// - no tab characters anywhere
// - keys are not indented; only array string elements use exactly 2-space indent
// - a blank line precedes each [section] header
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
