package features_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo user init — Create user starter files
// `silo user init` creates user-level starter files under `$XDG_CONFIG_HOME/silo/` if
// they do not already exist. It is idempotent: subsequent runs do not overwrite
// existing files.
func TestFeatureUserInit(t *testing.T) {
	// Background: the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: First run creates all user starter files", func(t *testing.T) {
		t.Run("Scenario: user init creates all three user files", func(t *testing.T) {
			base := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", base)

			// When I run `silo user init`
			err := cmd.UserInit([]string{})

			// Then a file "home-user.nix" should be created in the user's silo config directory
			// And a file "devcontainer.in.json" should be created in the user's silo config directory
			// And a file "silo.in.toml" should be created in the user's silo config directory
			siloDir := filepath.Join(base, "silo")
			for _, name := range []string{"home-user.nix", "devcontainer.in.json", "silo.in.toml"} {
				if _, err := os.Stat(filepath.Join(siloDir, name)); os.IsNotExist(err) {
					t.Errorf("expected %s to be created in user's silo config directory", name)
				}
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Idempotency — existing files are not overwritten", func(t *testing.T) {
		t.Run("Scenario: all existing user files are preserved", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "home-user.nix", "# custom content")
				internal.WriteUserFile(t, siloUser, "devcontainer.in.json", `{ "custom": true }`)
				internal.WriteUserFile(t, siloUser, "silo.in.toml", "[features]")
			})

			// When I run `silo user init`
			err := cmd.UserInit([]string{})

			// Then the file "home-user.nix" in the user's silo config directory should contain "# custom content"
			// And the file "devcontainer.in.json" in the user's silo config directory should contain "{ \"custom\": true }"
			// And the file "silo.in.toml" in the user's silo config directory should contain "[features]"
			xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
			siloDir := filepath.Join(xdgConfigHome, "silo")
			for _, tc := range []struct {
				name   string
				expect string
			}{
				{"home-user.nix", "# custom content"},
				{"devcontainer.in.json", `{ "custom": true }`},
				{"silo.in.toml", "[features]"},
			} {
				data, err := os.ReadFile(filepath.Join(siloDir, tc.name))
				if err != nil {
					t.Errorf("failed to read %s: %v", tc.name, err)
					continue
				}
				if string(data) != tc.expect {
					t.Errorf("expected %s to contain %q, got %q", tc.name, tc.expect, string(data))
				}
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Display of file status during user init", func(t *testing.T) {
		t.Run("Scenario: user init shows creating message for new files", func(t *testing.T) {
			base := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", base)

			// When I run `silo user init`
			output := internal.CaptureStdout(func() {
				cmd.UserInit([]string{})
			})

			// Then the output should contain "Creating <XDG_CONFIG_HOME>/silo/home-user.nix"
			// And the output should contain "Creating <XDG_CONFIG_HOME>/silo/devcontainer.in.json"
			// And the output should contain "Creating <XDG_CONFIG_HOME>/silo/silo.in.toml"
			expectedMsgs := []string{
				"Creating " + filepath.Join(base, "silo", "home-user.nix"),
				"Creating " + filepath.Join(base, "silo", "devcontainer.in.json"),
				"Creating " + filepath.Join(base, "silo", "silo.in.toml"),
			}
			for _, msg := range expectedMsgs {
				if !strings.Contains(output, msg) {
					t.Errorf("expected output to contain %q, got: %s", msg, output)
				}
			}
		})

		t.Run("Scenario: user init shows already exists message for existing files", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "home-user.nix", "{ config, pkgs, ... }:\n{\n}\n")
				internal.WriteUserFile(t, siloUser, "devcontainer.in.json", "{}\n")
				internal.WriteUserFile(t, siloUser, "silo.in.toml", "")
			})

			// When I run `silo user init`
			output := internal.CaptureStdout(func() {
				cmd.UserInit([]string{})
			})

			// Then the output should contain "'<XDG_CONFIG_HOME>/silo/home-user.nix' already exists"
			// And the output should contain "'<XDG_CONFIG_HOME>/silo/devcontainer.in.json' already exists"
			// And the output should contain "'<XDG_CONFIG_HOME>/silo/silo.in.toml' already exists"
			xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
			siloDir := filepath.Join(xdgConfigHome, "silo")
			expectedMsgs := []string{
				"'" + filepath.Join(siloDir, "home-user.nix") + "' already exists",
				"'" + filepath.Join(siloDir, "devcontainer.in.json") + "' already exists",
				"'" + filepath.Join(siloDir, "silo.in.toml") + "' already exists",
			}
			for _, msg := range expectedMsgs {
				if !strings.Contains(output, msg) {
					t.Errorf("expected output to contain %q, got: %s", msg, output)
				}
			}
		})
	})

	t.Run("Rule: --force overwrites existing user files", func(t *testing.T) {
		t.Run("Scenario: user init --force overwrites existing user files", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "home-user.nix", "# custom content")
				internal.WriteUserFile(t, siloUser, "devcontainer.in.json", `{ "custom": true }`)
				internal.WriteUserFile(t, siloUser, "silo.in.toml", "[features]")
			})

			// When I run `silo user init --force`
			if err := cmd.UserInit([]string{"--force"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the file "home-user.nix" in the user's silo config directory should contain the default content
			xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
			siloDir := filepath.Join(xdgConfigHome, "silo")
			for _, tc := range []struct {
				name        string
				expect      string
			}{
				{"home-user.nix", "config,\n  pkgs"},
				{"devcontainer.in.json", "{}\n"},
				{"silo.in.toml", ""},
			} {
				data, err := os.ReadFile(filepath.Join(siloDir, tc.name))
				if err != nil {
					t.Errorf("failed to read %s: %v", tc.name, err)
					continue
				}
				if tc.expect != "" && !strings.Contains(string(data), tc.expect) {
					t.Errorf("expected %s to contain %q after --force, got %q", tc.name, tc.expect, string(data))
				}
			}
			// And the exit code should be 0
		})
	})
}
