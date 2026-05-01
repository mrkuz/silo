package features_test

import (
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
			// Then a file "home-user.nix" should be created in the user's silo config directory
			// And a file "devcontainer.in.json" should be created in the user's silo config directory
			// And a file "silo.in.toml" should be created in the user's silo config directory
			userDir := filepath.Join(base, "silo")
			for _, name := range []string{"home-user.nix", "devcontainer.in.json", "silo.in.toml"} {
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
			internal.SubsequentRun(t, internal.MinimalConfig("abc12345"))

			// When I run `silo init`
			if err := cmd.Init([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the config should still have id "abc12345"
			saved, err := internal.ParseTOML(internal.SiloToml())
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if saved.General.ID != "abc12345" {
				t.Errorf("expected ID abc12345, got %q", saved.General.ID)
			}
			// And the exit code should be 0 (implicit)
		})

		t.Run("Scenario: existing shared-volume and podman settings are preserved when flags not provided", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			// And the config has shared_volume=true and podman=true
			cfg := internal.MinimalConfig("abc12345")
			cfg.Features.SharedVolume = true
			cfg.Features.Podman = true
			internal.SubsequentRun(t, cfg)

			// When I run `silo init`
			if err := cmd.Init([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the config should still have shared_volume=true
			// And the config should still have podman=true
			saved, err := internal.ParseTOML(internal.SiloToml())
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if !saved.Features.SharedVolume {
				t.Error("expected SharedVolume to remain true")
			}
			if !saved.Features.Podman {
				t.Error("expected Podman to remain true")
			}
			// And the exit code should be 0 (implicit)
		})
	})

	t.Run("Rule: silo.in.toml seeds new workspace config on first run", func(t *testing.T) {
		t.Run("Scenario: silo.in.toml values seed the workspace config", func(t *testing.T) {
			// Given the user's silo config directory has "silo.in.toml" with:
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.in.toml", `
					[features]
					shared_volume = true
					podman = true

					[shared_volume]
					name = "my-shared"
					paths = ["$HOME/.cache/uv/"]

					[create]
					arguments = ["--memory=2g"]
				`)
			})

			// When I run `silo init`
			if err := cmd.Init([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			saved, err := internal.ParseTOML(internal.SiloToml())
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			// Then the workspace config should have shared_volume=true
			if !saved.Features.SharedVolume {
				t.Error("expected SharedVolume=true from silo.in.toml")
			}
			// And the workspace config should have podman=true
			if !saved.Features.Podman {
				t.Error("expected Podman=true from silo.in.toml")
			}
			// And the workspace config should have shared_volume name "my-shared"
			if saved.SharedVolume.Name != "my-shared" {
				t.Errorf("expected SharedVolume.Name=\"my-shared\", got %q", saved.SharedVolume.Name)
			}
			// And the workspace config should have create arguments ["--memory=2g"]
			if len(saved.Create.Arguments) < 2 {
				t.Errorf("expected at least 2 create arguments, got %v", saved.Create.Arguments)
			}
			if saved.Create.Arguments[0] != "--memory=2g" {
				t.Errorf("expected first argument --memory=2g, got %v", saved.Create.Arguments)
			}
		})

		t.Run("Scenario: silo.in.toml [general] section is ignored", func(t *testing.T) {
			// Given the user's silo config directory has "silo.in.toml" with:
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.in.toml", `
					[general]
					id = "ignored-id"
					user = "ignored-user"
					container_name = "ignored-container"
					image_name = "ignored-image"
				`)
			})

			// When I run `silo init`
			if err := cmd.Init([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			saved, err := internal.ParseTOML(internal.SiloToml())
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			// Then the workspace config should have an 8-character random id
			if len(saved.General.ID) != 8 {
				t.Errorf("expected 8-character random id, got %q", saved.General.ID)
			}
			// And the workspace config should use the current username
			u, _ := user.Current()
			if saved.General.User != u.Username {
				t.Errorf("expected current user %q, got %q", u.Username, saved.General.User)
			}
			// And the workspace config should have container_name starting with "silo-"
			if !strings.HasPrefix(saved.General.ContainerName, "silo-") {
				t.Errorf("expected container_name starting with \"silo-\", got %q", saved.General.ContainerName)
			}
		})

		t.Run("Scenario: silo.in.toml empty or absent uses built-in defaults", func(t *testing.T) {
			// Given the user's silo config directory has "silo.in.toml" with:
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.in.toml", "")
			})

			// When I run `silo init`
			if err := cmd.Init([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			saved, err := internal.ParseTOML(internal.SiloToml())
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			// Then the workspace config should have shared_volume=false
			if saved.Features.SharedVolume {
				t.Error("expected SharedVolume=false by default")
			}
			// And the workspace config should have podman=false
			if saved.Features.Podman {
				t.Error("expected Podman=false by default")
			}
			// And the workspace config should have shared_volume name "silo-shared"
			if saved.SharedVolume.Name != "silo-shared" {
				t.Errorf("expected default SharedVolume.Name=\"silo-shared\", got %q", saved.SharedVolume.Name)
			}
		})

		t.Run("Scenario: silo.in.toml is created if it does not exist", func(t *testing.T) {
			// Given the user's silo config directory exists but "silo.in.toml" is absent
			base := internal.FirstRunWith(t, nil) // nil configFunc = don't write silo.in.toml

			// When I run `silo init`
			if err := cmd.Init([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			userTomlPath := filepath.Join(base, "silo", "silo.in.toml")
			// Then a file "silo.in.toml" should be created in the user's silo config directory
			if _, err := os.Stat(userTomlPath); os.IsNotExist(err) {
				t.Error("expected silo.in.toml to be created in user's config directory")
			}
			content, err := os.ReadFile(userTomlPath)
			if err != nil {
				t.Fatalf("failed to read silo.in.toml: %v", err)
			}
			// And the file "silo.in.toml" in the user's silo config directory should be empty
			if len(content) != 0 {
				t.Errorf("expected empty silo.in.toml, got %q", string(content))
			}
		})

		t.Run("Scenario: silo.in.toml create arguments are prepended to default arguments", func(t *testing.T) {
			// Given the user's silo config directory has "silo.in.toml" with:
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.in.toml", `
				[create]
				arguments = ["--memory=2g"]
				`)
			})

			// When I run `silo init`
			if err := cmd.Init([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			saved, err := internal.ParseTOML(internal.SiloToml())
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			// Then the workspace config should have 5 create arguments
			if len(saved.Create.Arguments) != 5 {
				t.Errorf("expected 5 create arguments, got %v", saved.Create.Arguments)
			}
			// And the first create argument should be "--memory=2g"
			if saved.Create.Arguments[0] != "--memory=2g" {
				t.Errorf("expected first create argument --memory=2g, got %v", saved.Create.Arguments)
			}
			// And the second create argument should be "--cap-drop=ALL"
			if saved.Create.Arguments[1] != "--cap-drop=ALL" {
				t.Errorf("expected second create argument --cap-drop=ALL, got %v", saved.Create.Arguments)
			}
		})
	})

	t.Run("Rule: Explicit flags override existing config", func(t *testing.T) {
		t.Run("Scenario: --podman enables podman feature", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			// And the user's XDG_CONFIG_HOME points to a fresh directory
			internal.FirstRun(t)

			// When I run `silo init --podman`
			if err := cmd.Init([]string{"--podman"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the config should have podman=true
			saved, err := internal.ParseTOML(internal.SiloToml())
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if !saved.Features.Podman {
				t.Error("expected Features.Podman=true after --podman flag on first run")
			}
			// And the exit code should be 0 (implicit)
		})

		t.Run("Scenario: --no-podman disables podman feature", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			// And the user's XDG_CONFIG_HOME points to a fresh directory
			internal.FirstRun(t)

			// When I run `silo init --no-podman`
			if err := cmd.Init([]string{"--no-podman"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the config should have podman=false
			saved, err := internal.ParseTOML(internal.SiloToml())
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if saved.Features.Podman {
				t.Error("expected Features.Podman=false after --no-podman flag on first run")
			}
			// And the exit code should be 0 (implicit)
		})

		t.Run("Scenario: --shared-volume enables shared volume", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			// And the config has shared_volume=false
			cfg := internal.MinimalConfig("abc12345")
			cfg.Features.SharedVolume = false
			internal.SubsequentRun(t, cfg)

			// When I run `silo init --shared-volume`
			if err := cmd.Init([]string{"--shared-volume"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the config should have shared_volume=true
			saved, err := internal.ParseTOML(internal.SiloToml())
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if !saved.Features.SharedVolume {
				t.Error("expected Features.SharedVolume=true after --shared-volume flag")
			}
			// And the exit code should be 0 (implicit)
		})

		t.Run("Scenario: --no-shared-volume disables shared volume", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			// And the config has shared_volume=true
			cfg := internal.MinimalConfig("abc12345")
			cfg.Features.SharedVolume = true
			internal.SubsequentRun(t, cfg)

			// When I run `silo init --no-shared-volume`
			if err := cmd.Init([]string{"--no-shared-volume"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the config should have shared_volume=false
			saved, err := internal.ParseTOML(internal.SiloToml())
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if saved.Features.SharedVolume {
				t.Error("expected Features.SharedVolume=false after --no-shared-volume flag")
			}
			// And the exit code should be 0 (implicit)
		})

		t.Run("Scenario: --podman flag overrides seeded config from silo.in.toml", func(t *testing.T) {
			// Given the user's silo config directory has "silo.in.toml" with:
			// And a clean workspace with no existing silo files
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.in.toml", `
				[features]
				podman = true
				`)
			})

			// When I run `silo init --no-podman`
			if err := cmd.Init([]string{"--no-podman"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the workspace config should have podman=false
			saved, err := internal.ParseTOML(internal.SiloToml())
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if saved.Features.Podman {
				t.Error("expected Features.Podman=false after --no-podman flag overriding seeded true")
			}
			// And the exit code should be 0 (implicit)
		})

		t.Run("Scenario: --shared-volume flag overrides seeded config from silo.in.toml", func(t *testing.T) {
			// Given the user's silo config directory has "silo.in.toml" with:
			// And a clean workspace with no existing silo files
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.in.toml", `
				[features]
				shared_volume = true
				`)
			})

			// When I run `silo init --no-shared-volume`
			if err := cmd.Init([]string{"--no-shared-volume"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the workspace config should have shared_volume=false
			saved, err := internal.ParseTOML(internal.SiloToml())
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if saved.Features.SharedVolume {
				t.Error("expected Features.SharedVolume=false after --no-shared-volume flag overriding seeded true")
			}
			// And the exit code should be 0 (implicit)
		})
	})

	t.Run("Rule: Podman flag affects workspace home.nix", func(t *testing.T) {
		t.Run("Scenario: --podman adds podman module to home.nix", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			internal.FirstRun(t)

			// When I run `silo init --podman`
			if err := cmd.Init([]string{"--podman"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the file ".silo/home.nix" should contain "module.podman.enable = true"
			content, err := os.ReadFile(internal.SiloDir() + "/home.nix")
			if err != nil {
				t.Fatalf("failed to read .silo/home.nix: %v", err)
			}
			if !strings.Contains(string(content), "module.podman.enable = true") {
				t.Errorf("expected 'module.podman.enable = true' in home.nix, got: %s", content)
			}
		})

		t.Run("Scenario: --no-podman does not add podman module to home.nix", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			internal.FirstRun(t)

			// When I run `silo init --no-podman`
			if err := cmd.Init([]string{"--no-podman"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the file ".silo/home.nix" should not contain "module.podman.enable = true"
			content, err := os.ReadFile(internal.SiloDir() + "/home.nix")
			if err != nil {
				t.Fatalf("failed to read .silo/home.nix: %v", err)
			}
			if strings.Contains(string(content), "module.podman.enable = true") {
				t.Errorf("expected no 'module.podman.enable = true' in home.nix with --no-podman, got: %s", content)
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
			saved, err := internal.ParseTOML(internal.SiloToml())
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if saved.Features.Podman {
				t.Error("expected Features.Podman=false when both --podman and --no-podman passed")
			}
			// And the exit code should be 0 (implicit)
		})

		t.Run("Scenario: both --shared-volume and --no-shared-volume uses last flag", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			internal.FirstRun(t)

			// When I run `silo init --shared-volume --no-shared-volume`
			if err := cmd.Init([]string{"--shared-volume", "--no-shared-volume"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then the config should have shared_volume=false
			saved, err := internal.ParseTOML(internal.SiloToml())
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if saved.Features.SharedVolume {
				t.Error("expected Features.SharedVolume=false when both --shared-volume and --no-shared-volume passed")
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
			internal.SubsequentRun(t, internal.MinimalConfig("abc12345"))

			// When I run `silo init`
			output := internal.CaptureStdout(func() { cmd.Init([]string{}) })

			// Then the output should contain "'/path/to/workspace/.silo/silo.toml' already exists"
			if !strings.Contains(output, "already exists") {
				t.Errorf("expected output to contain 'already exists' message, got: %s", output)
			}
		})
	})
}
