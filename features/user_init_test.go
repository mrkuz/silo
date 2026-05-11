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

			// Then a file "home.user.nix" should be created in the user's silo config directory
			// And a file "devcontainer.user.json" should be created in the user's silo config directory
			// And a file "silo.user.toml" should be created in the user's silo config directory
			siloDir := filepath.Join(base, "silo")
			for _, name := range []string{"home.user.nix", "devcontainer.user.json", "silo.user.toml"} {
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

	t.Run("Scenario: home.user.nix contains default shell command module", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)

		if err := cmd.UserInit([]string{}); err != nil {
			t.Fatalf("expected exit code 0, got error: %v", err)
		}

		siloDir := filepath.Join(base, "silo")
		data, err := os.ReadFile(filepath.Join(siloDir, "home.user.nix"))
		if err != nil {
			t.Fatalf("failed to read home.user.nix: %v", err)
		}
		content := string(data)
		if !strings.Contains(content, "silo.shellCommand") {
			t.Errorf("expected home.user.nix to contain silo.shellCommand, got: %s", content)
		}
		if !strings.Contains(content, "/bin/bash --login") {
			t.Errorf("expected home.user.nix to contain /bin/bash --login, got: %s", content)
		}
	})

	t.Run("Scenario: silo.user.toml contains [general] with current username", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		u, _ := user.Current()

		// When I run `silo user init`
		if err := cmd.UserInit([]string{}); err != nil {
			t.Fatalf("expected exit code 0, got error: %v", err)
		}

		siloDir := filepath.Join(base, "silo")
		data, err := os.ReadFile(filepath.Join(siloDir, "silo.user.toml"))
		if err != nil {
			t.Fatalf("failed to read silo.user.toml: %v", err)
		}
		// Then the file "silo.user.toml" in the user's silo config directory should contain `[general]`
		if !bytes.Contains(data, []byte("[general]")) {
			t.Errorf("expected silo.user.toml to contain [general], got: %s", string(data))
		}
		// And the file "silo.user.toml" in the user's silo config directory should contain the current username
		if !bytes.Contains(data, []byte(u.Username)) {
			t.Errorf("expected silo.user.toml to contain username %q, got: %s", u.Username, string(data))
		}
		// And the exit code should be 0
		if err != nil {
			t.Errorf("expected exit code 0, got error: %v", err)
		}
	})

	t.Run("Rule: Idempotency — existing files are not overwritten", func(t *testing.T) {
		t.Run("Scenario: all existing user files are preserved", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "home.user.nix", "# custom content")
				internal.WriteUserFile(t, siloUser, "devcontainer.user.json", `{ "custom": true }`)
				internal.WriteUserFile(t, siloUser, "silo.user.toml", `[general]
user = "testuser"

[persistence]
shared_paths = []
`)
			})

			// When I run `silo user init`
			err := cmd.UserInit([]string{})

			// Then the file "home.user.nix" in the user's silo config directory should contain "# custom content"
			// And the file "devcontainer.user.json" in the user's silo config directory should contain "{ \"custom\": true }"
			// And the file "silo.user.toml" in the user's silo config directory should contain "[general]"
			xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
			siloDir := filepath.Join(xdgConfigHome, "silo")
			for _, tc := range []struct {
				name    string
				partial string
			}{
				{"home.user.nix", "# custom content"},
				{"devcontainer.user.json", `{ "custom": true }`},
				{"silo.user.toml", "[general]"},
			} {
				data, err := os.ReadFile(filepath.Join(siloDir, tc.name))
				if err != nil {
					t.Errorf("failed to read %s: %v", tc.name, err)
					continue
				}
				if !strings.Contains(string(data), tc.partial) {
					t.Errorf("expected %s to contain %q, got %q", tc.name, tc.partial, string(data))
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

			// Then the output should contain "Creating <XDG_CONFIG_HOME>/silo/home.user.nix"
			// And the output should contain "Creating <XDG_CONFIG_HOME>/silo/devcontainer.user.json"
			// And the output should contain "Creating <XDG_CONFIG_HOME>/silo/silo.user.toml"
			expectedMsgs := []string{
				"Creating " + filepath.Join(base, "silo", "home.user.nix"),
				"Creating " + filepath.Join(base, "silo", "devcontainer.user.json"),
				"Creating " + filepath.Join(base, "silo", "silo.user.toml"),
			}
			for _, msg := range expectedMsgs {
				if !strings.Contains(output, msg) {
					t.Errorf("expected output to contain %q, got: %s", msg, output)
				}
			}
		})

		t.Run("Scenario: user init shows already exists message for existing files", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "home.user.nix", "{ config, pkgs, ... }:\n{\n}\n")
				internal.WriteUserFile(t, siloUser, "devcontainer.user.json", "{}\n")
				internal.WriteUserFile(t, siloUser, "silo.user.toml", "")
			})

			// When I run `silo user init`
			output := internal.CaptureStdout(func() {
				cmd.UserInit([]string{})
			})

			// Then the output should contain "'<XDG_CONFIG_HOME>/silo/home.user.nix' already exists"
			// And the output should contain "'<XDG_CONFIG_HOME>/silo/devcontainer.user.json' already exists"
			// And the output should contain "'<XDG_CONFIG_HOME>/silo/silo.user.toml' already exists"
			xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
			siloDir := filepath.Join(xdgConfigHome, "silo")
			expectedMsgs := []string{
				"'" + filepath.Join(siloDir, "home.user.nix") + "' already exists",
				"'" + filepath.Join(siloDir, "devcontainer.user.json") + "' already exists",
				"'" + filepath.Join(siloDir, "silo.user.toml") + "' already exists",
			}
			for _, msg := range expectedMsgs {
				if !strings.Contains(output, msg) {
					t.Errorf("expected output to contain %q, got: %s", msg, output)
				}
			}
		})
	})

	t.Run("Rule: Requires valid user config", func(t *testing.T) {
		t.Run("Scenario: missing user in silo.user.toml returns error", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.user.toml", "[general]\n")
			})

			// When I run `silo user init`
			err := cmd.UserInit([]string{})

			// Then the exit code should not be 0
			if err == nil {
				t.Fatal("expected error when user is missing")
			}
			// And the error should indicate "[general].user is required"
			if !strings.Contains(err.Error(), "[general].user is required") {
				t.Errorf("expected error to mention '[general].user is required', got: %v", err)
			}
		})
	})
}
