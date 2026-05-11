package features_test

import (
	"bytes"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo init — Initialize workspace
// `silo init` creates workspace configuration and starter files. It is idempotent:
// subsequent runs do not overwrite existing files.
func TestFeatureInit(t *testing.T) {
	// Background: a clean workspace with no existing silo files
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: First run creates workspace files", func(t *testing.T) {
		t.Run("Scenario: init creates .silo directory with config and home.nix", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			// And the user's XDG_CONFIG_HOME points to a fresh directory
			internal.FirstRun(t)

			// When I run `silo init`
			if err := cmd.Init([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then a file ".silo/silo.toml" should be created
			if _, err := os.Stat(internal.SiloToml()); os.IsNotExist(err) {
				t.Error("expected .silo/silo.toml to be created")
			}
			// And a file ".silo/home.nix" should be created
			if _, err := os.Stat(internal.SiloDir() + "/home.nix"); os.IsNotExist(err) {
				t.Error("expected .silo/home.nix to be created")
			}
			// And the exit code should be 0 (implicit)
		})

		t.Run("Scenario: init creates user starter files", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			// And the user's XDG_CONFIG_HOME points to a fresh directory
			base := internal.FirstRun(t)

			// When I run `silo init`
			if err := cmd.Init([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then a file "home.user.nix" should be created in the user's silo config directory
			// And a file "devcontainer.user.json" should be created in the user's silo config directory
			// And a file "silo.user.toml" should be created in the user's silo config directory
			userDir := filepath.Join(base, "silo")
			for _, name := range []string{"home.user.nix", "devcontainer.user.json", "silo.user.toml"} {
				if _, err := os.Stat(filepath.Join(userDir, name)); os.IsNotExist(err) {
					t.Errorf("expected %s to be created in user's silo config directory", name)
				}
			}
			// And the exit code should be 0 (implicit)
		})
	})

	t.Run("Rule: Idempotency — subsequent runs do not modify existing config", func(t *testing.T) {
		t.Run("Scenario: existing config is not overwritten", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			// And the config has id "abc12345"
			internal.SubsequentRun(t, internal.MinimalConfig("abc12345"), "alice")

			// When I run `silo init`
			if err := cmd.Init([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the config should still have id "abc12345"
			var saved internal.WorkspaceConfig
			if err := internal.ParseTOML(internal.SiloToml(), &saved); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if saved.General.ID != "abc12345" {
				t.Errorf("expected ID abc12345, got %q", saved.General.ID)
			}
			// And the exit code should be 0 (implicit)
		})

		t.Run("Scenario: unknown flag shows error and help", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			internal.FirstRun(t)

			// When I run `silo init --unknown`
			err := cmd.Init([]string{"--unknown"})

			// Then the exit code should not be 0
			if err == nil {
				t.Fatal("expected error for unknown flag")
			}
			// And the error should contain "erroneous command"
			if !strings.Contains(err.Error(), `erroneous command`) {
				t.Errorf("expected erroneous command error, got: %v", err)
			}
			// And the error should contain the help text
			if !strings.Contains(err.Error(), "Usage:") {
				t.Errorf("expected help text in error, got: %v", err)
			}
		})

		t.Run("Scenario: existing shared-volume and podman settings are preserved when flags not provided", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			// And the config has shared_paths set and podman=true
			cfg := internal.MinimalConfig("abc12345")
			cfg.Persistence.SharedPaths = []string{"$HOME/.cache/uv/"}
			cfg.Features.Podman = true
			internal.SubsequentRun(t, cfg, "alice")

			// When I run `silo init`
			if err := cmd.Init([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the config should still have shared_paths set
			// And the config should still have podman=true
			var saved internal.WorkspaceConfig
			if err := internal.ParseTOML(internal.SiloToml(), &saved); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if len(saved.Persistence.SharedPaths) == 0 {
				t.Error("expected Persistence.SharedPaths to remain set")
			}
			if !saved.Features.Podman {
				t.Error("expected Podman to remain true")
			}
			// And the exit code should be 0 (implicit)
		})

		t.Run("Scenario: existing podman setting is preserved when flag provided", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.Features.Podman = true
			internal.SubsequentRun(t, cfg, "alice")

			if err := cmd.Init([]string{"--no-podman"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var saved internal.WorkspaceConfig
			if err := internal.ParseTOML(internal.SiloToml(), &saved); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if !saved.Features.Podman {
				t.Error("expected Podman to remain true after --no-podman on subsequent run")
			}
		})
	})

	t.Run("Rule: silo.init creates workspace config with defaults", func(t *testing.T) {
		t.Run("Scenario: init creates workspace config with defaults", func(t *testing.T) {
			// Given the user's silo config directory has "silo.user.toml" with:
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.user.toml", `
					[general]
					user = "alice"

					[persistence]
					shared_paths = ["$HOME/.cache/uv/"]

					[podman]
					create_args = ["--memory=2g"]
				`)
			})

			// When I run `silo init`
			if err := cmd.Init([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var saved internal.WorkspaceConfig
			if err := internal.ParseTOML(internal.SiloToml(), &saved); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			// Then the workspace config should have defaults with no user (merged at read time)
			if saved.General.ID == "" {
				t.Error("expected ID to be set")
			}
			// And the workspace config should have podman=false (default)
			if saved.Features.Podman {
				t.Error("expected podman=false (default)")
			}
			// And the workspace config should have empty shared_paths (default)
			if len(saved.Persistence.SharedPaths) != 0 {
				t.Errorf("expected empty paths, got %v", saved.Persistence.SharedPaths)
			}
			// And the workspace config should have default create_args (not empty)
			if len(saved.Podman.CreateArgs) == 0 {
				t.Error("expected default create_args (not empty)")
			}
			// And the workspace config should have an 8-character random id
			if len(saved.General.ID) != 8 {
				t.Errorf("expected 8-character random id, got %q", saved.General.ID)
			}
		})

		t.Run("Scenario: silo.user.toml is created if it does not exist", func(t *testing.T) {
			// Given the user's silo config directory exists but "silo.user.toml" is absent
			base := internal.FirstRunWith(t, nil) // nil configFunc = don't write silo.user.toml
			u, _ := user.Current()

			// When I run `silo init`
			if err := cmd.Init([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			userTomlPath := filepath.Join(base, "silo", "silo.user.toml")
			// Then a file "silo.user.toml" should be created in the user's silo config directory
			if _, err := os.Stat(userTomlPath); os.IsNotExist(err) {
				t.Error("expected silo.user.toml to be created in user's config directory")
			}
			content, err := os.ReadFile(userTomlPath)
			if err != nil {
				t.Fatalf("failed to read silo.user.toml: %v", err)
			}
			// And the file "silo.user.toml" in the user's silo config directory should contain `[general]`
			if !bytes.Contains(content, []byte("[general]")) {
				t.Errorf("expected silo.user.toml to contain [general], got %q", string(content))
			}
			// And the file "silo.user.toml" in the user's silo config directory should contain the current username
			if !bytes.Contains(content, []byte(u.Username)) {
				t.Errorf("expected silo.user.toml to contain username %q, got %q", u.Username, string(content))
			}
		})
	})

	t.Run("Rule: Feature flags set initial config on first run", func(t *testing.T) {
		t.Run("Scenario: --podman sets podman=true on first run", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			internal.FirstRun(t)

			// When I run `silo init --podman`
			if err := cmd.Init([]string{"--podman"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the workspace config should have podman=true
			var saved internal.WorkspaceConfig
			if err := internal.ParseTOML(internal.SiloToml(), &saved); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if !saved.Features.Podman {
				t.Error("expected Features.Podman=true after --podman on first run")
			}
			// And the file ".silo/home.nix" should contain "silo.podman.enable = true"
			content, err := os.ReadFile(internal.SiloDir() + "/home.nix")
			if err != nil {
				t.Fatalf("failed to read .silo/home.nix: %v", err)
			}
			if !strings.Contains(string(content), "silo.podman.enable = true") {
				t.Errorf("expected 'silo.podman.enable = true' in home.nix, got: %s", content)
			}
		})

		t.Run("Scenario: --no-podman sets podman=false on first run", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			internal.FirstRun(t)

			// When I run `silo init --no-podman`
			if err := cmd.Init([]string{"--no-podman"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the workspace config should have podman=false
			var saved internal.WorkspaceConfig
			if err := internal.ParseTOML(internal.SiloToml(), &saved); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if saved.Features.Podman {
				t.Error("expected Features.Podman=false after --no-podman on first run")
			}
			// And the file ".silo/home.nix" should contain "silo.podman.enable = false"
			content, err := os.ReadFile(internal.SiloDir() + "/home.nix")
			if err != nil {
				t.Fatalf("failed to read .silo/home.nix: %v", err)
			}
			if !strings.Contains(string(content), "silo.podman.enable = false") {
				t.Errorf("expected 'silo.podman.enable = false' in home.nix with --no-podman, got: %s", content)
			}
		})
	})

	t.Run("Rule: Conflicting flags use last value", func(t *testing.T) {
		t.Run("Scenario: both --podman and --no-podman uses last flag", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			internal.FirstRun(t)

			// When I run `silo init --podman --no-podman`
			if err := cmd.Init([]string{"--podman", "--no-podman"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the config should have podman=false
			var saved internal.WorkspaceConfig
			if err := internal.ParseTOML(internal.SiloToml(), &saved); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if saved.Features.Podman {
				t.Error("expected Features.Podman=false when both --podman and --no-podman passed")
			}
			// And the exit code should be 0 (implicit)
		})
	})

	t.Run("Rule: Display of file status during init", func(t *testing.T) {
		t.Run("Scenario: init shows creating message for new files", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			internal.FirstRun(t)

			// When I run `silo init`
			output := internal.CaptureStdout(func() { cmd.Init([]string{}) })

			// Then the output should contain "Creating .silo/silo.toml"
			if !strings.Contains(output, "Creating .silo/silo.toml") {
				t.Errorf("expected output to contain 'Creating .silo/silo.toml', got: %s", output)
			}
			// And the output should contain "Creating .silo/home.nix"
			if !strings.Contains(output, "Creating .silo/home.nix") {
				t.Errorf("expected output to contain 'Creating .silo/home.nix', got: %s", output)
			}
		})

		t.Run("Scenario: init shows already exists message for existing files", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			internal.SubsequentRun(t, internal.MinimalConfig("abc12345"), "alice")

			// When I run `silo init`
			output := internal.CaptureStdout(func() { cmd.Init([]string{}) })

			// Then the output should contain "'/path/to/workspace/.silo/silo.toml' already exists"
			if !strings.Contains(output, "already exists") {
				t.Errorf("expected output to contain 'already exists' message, got: %s", output)
			}
		})
	})

	t.Run("Rule: Requires workspace config to be valid", func(t *testing.T) {
		t.Run("Scenario: missing id in .silo/silo.toml returns error", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.ID = ""
			internal.SubsequentRun(t, cfg, "alice")

			// When I run `silo init`
			err := cmd.Init([]string{})

			// Then the exit code should not be 0
			if err == nil {
				t.Fatal("expected error when id is empty")
			}
			// And the error should indicate ".silo/silo.toml" is missing required field
			if !strings.Contains(err.Error(), ".silo/silo.toml") {
				t.Errorf("expected error to mention '.silo/silo.toml', got: %v", err)
			}
		})
	})
}
