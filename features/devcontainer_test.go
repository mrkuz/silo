package features_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo devcontainer — Generate a .devcontainer.json for VS Code
// `silo devcontainer` generates a `.devcontainer.json` for VS Code in the current
// directory. It is independent from the main workspace container (silo-<id>) and is
// managed separately by VS Code. The generated devcontainer uses the workspace image
// and the container name is `<workspace-container-name>-dev`.
func TestFeatureDevcontainer(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory
	// and the user's silo config directory has all starter files

	t.Run("Rule: Generates .devcontainer.json", func(t *testing.T) {
		t.Run("Scenario: devcontainer generates a .devcontainer.json file", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)
			internal.SetupUserConfig(t)

			// When I run `silo devcontainer`
			err := cmd.DevcontainerGenerate([]string{})

			// Then a file ".devcontainer.json" should be created
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, statErr := os.Stat(".devcontainer.json"); os.IsNotExist(statErr) {
				t.Errorf(".devcontainer.json was not created")
			}
			// And the output should contain "Generated .devcontainer.json"
			// And the exit code should be 0
		})

		t.Run("Scenario: existing .devcontainer.json is not overwritten", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)
			internal.SetupUserConfig(t)

			// Given a file ".devcontainer.json" already exists with content '{"name": "custom"}'
			existing := []byte(`{"name": "custom"}`)
			if err := os.WriteFile(".devcontainer.json", existing, 0644); err != nil {
				t.Fatal(err)
			}

			// When I run `silo devcontainer`
			output := internal.CaptureStdout(func() { cmd.DevcontainerGenerate([]string{}) })

			// Then the file ".devcontainer.json" should still contain '{"name": "custom"}'
			got, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(existing) {
				t.Errorf("expected existing file to be preserved, got %s", string(got))
			}
			// And the output should contain ".devcontainer.json already exists"
			if !strings.Contains(output, "'.devcontainer.json' already exists") {
				t.Errorf("expected 'already exists' in output, got: %s", output)
			}
		})

		t.Run("Scenario: --force overwrites existing .devcontainer.json", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)
			internal.SetupUserConfig(t)

			// Given a file ".devcontainer.json" already exists with content '{"name": "custom"}'
			existing := []byte(`{"name": "custom"}`)
			if err := os.WriteFile(".devcontainer.json", existing, 0644); err != nil {
				t.Fatal(err)
			}

			// When I run `silo devcontainer --force`
			output := internal.CaptureStdout(func() { cmd.DevcontainerGenerate([]string{"--force"}) })

			// Then the output should contain ".devcontainer.json updated"
			if !strings.Contains(output, "'.devcontainer.json' already exists - overwritten") {
				t.Errorf("expected '.devcontainer.json' updated in output, got: %s", output)
			}

			// Then the file ".devcontainer.json" should not contain '{"name": "custom"}'
			got, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatal(err)
			}
			if string(got) == string(existing) {
				t.Errorf("expected file to be overwritten, still contains custom content")
			}
		})

		t.Run("Scenario: --force does not affect .silo/devcontainer.json handling", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)
			internal.SetupUserConfig(t)

			// Given a file ".silo/devcontainer.json" already exists with content '{"custom": true}'
			if err := os.MkdirAll(".silo", 0755); err != nil {
				t.Fatal(err)
			}
			existing := []byte(`{"custom": true}`)
			if err := os.WriteFile(".silo/devcontainer.json", existing, 0644); err != nil {
				t.Fatal(err)
			}

			// When I run `silo devcontainer --force`
			output := internal.CaptureStdout(func() { cmd.DevcontainerGenerate([]string{"--force"}) })

			// Then the output should contain "'.silo/devcontainer.json' already exists"
			if !strings.Contains(output, "'.silo/devcontainer.json' already exists") {
				t.Errorf("expected 'already exists' in output, got: %s", output)
			}
			// And a file ".silo/devcontainer.json" should still contain '{"custom": true}'
			got, err := os.ReadFile(".silo/devcontainer.json")
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(existing) {
				t.Errorf("expected file to be preserved, got %s", string(got))
			}
		})

		t.Run("Scenario: devcontainer uses the workspace image", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)
			internal.SetupUserConfig(t)

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the .devcontainer.json should reference image "silo-abc12345"
			data, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatalf("read .devcontainer.json: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("expected valid json: %v", err)
			}
			if name, ok := parsed["image"].(string); ok {
				if name != "silo-abc12345" {
					t.Errorf("expected image name 'silo-abc12345', got: %s", name)
				}
			}
		})

		t.Run("Scenario: devcontainer uses a distinct container name", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)
			internal.SetupUserConfig(t)

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the .devcontainer.json should specify container name "silo-abc12345-dev"
			data, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatalf("read .devcontainer.json: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("expected valid json: %v", err)
			}
			if name, ok := parsed["name"].(string); ok {
				if name != "silo-abc12345-dev" {
					t.Errorf("expected container name 'silo-abc12345-dev', got: %s", name)
				}
			}
		})

		t.Run("Scenario: unknown flag shows error and help", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)
			internal.SetupUserConfig(t)

			// When I run `silo devcontainer --unknown`
			err := cmd.DevcontainerGenerate([]string{"--unknown"})

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

		t.Run("Scenario: devcontainer runs volume setup before generating when shared volume is configured", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
			internal.SubsequentRun(t, cfg, "testuser")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-testuser":     exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman run --rm <...>":                 exec.Command("true"),
			})

			// When I run `silo devcontainer`
			cmd.DevcontainerGenerate([]string{})

			// Then shared volume directories should be created before generating .devcontainer.json.
			record := mock.AssertExec("podman", "run", "--rm", "<...>")
			cmdStr := strings.Join(record.Args, " ")
			expectedPath := "/silo/shared/home/testuser/.cache/uv"
			if !strings.Contains(cmdStr, "mkdir -p "+expectedPath) {
				t.Errorf("expected volume setup with 'mkdir -p %s', got: %s", expectedPath, cmdStr)
			}
		})
	})

	t.Run("Rule: Merge order: template wins > .silo > user", func(t *testing.T) {
		t.Run("Scenario: template values override user config", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.user.toml", `[general]
user = "alice"
`)
				internal.WriteUserFile(t, siloUser, "devcontainer.user.json", `{"name": "my-devcontainer"}`)
			})
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the .devcontainer.json should have "name" from TEMPLATE (not user)
			data, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatalf("read .devcontainer.json: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("expected valid json: %v", err)
			}
			if name, ok := parsed["name"].(string); ok {
				if name != "silo-abc12345-dev" {
					t.Errorf("expected template name 'silo-abc12345-dev', got: %s", name)
				}
			} else {
				t.Errorf("expected 'name' to be set from template")
			}
		})

		t.Run("Scenario: .silo overrides user config, template wins over both", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.user.toml", `[general]
user = "alice"
`)
				internal.WriteUserFile(t, siloUser, "devcontainer.user.json", `{"name": "user-name", "custom": "user-value"}`)
			})
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)

			// Given .silo/devcontainer.json has different values
			if err := os.MkdirAll(".silo", 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(".silo/devcontainer.json", []byte(`{"name": "silo-name", "custom": "silo-value"}`), 0644); err != nil {
				t.Fatal(err)
			}

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then template should win for "name", .silo should win for "custom" (not in template)
			data, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatalf("read .devcontainer.json: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("expected valid json: %v", err)
			}
			if name := parsed["name"].(string); name != "silo-abc12345-dev" {
				t.Errorf("expected template name 'silo-abc12345-dev', got: %s", name)
			}
			if custom := parsed["custom"].(string); custom != "silo-value" {
				t.Errorf("expected .silo value 'silo-value' for 'custom', got: %s", custom)
			}
		})

		t.Run("Scenario: arrays from all sources are concatenated", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.user.toml", `[general]
user = "alice"
`)
				internal.WriteUserFile(t, siloUser, "devcontainer.user.json", `{"features": ["user-feat"]}`)
			})
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)

			// Given .silo/devcontainer.json has features
			if err := os.MkdirAll(".silo", 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(".silo/devcontainer.json", []byte(`{"features": ["silo-feat"]}`), 0644); err != nil {
				t.Fatal(err)
			}

			// Template doesn't have features, so we need to set it there
			// Actually template has no features, so the first merge (user) sets ["user-feat"]
			// Then .silo concat to ["user-feat", "silo-feat"]

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then features should be concatenated: user first, then .silo
			data, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatalf("read .devcontainer.json: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("expected valid json: %v", err)
			}
			features, ok := parsed["features"].([]any)
			if !ok {
				t.Fatalf("expected features to be array, got: %v", parsed["features"])
			}
			if len(features) != 2 {
				t.Errorf("expected 2 features, got %d: %v", len(features), features)
			}
			if features[0] != "user-feat" || features[1] != "silo-feat" {
				t.Errorf("expected ['user-feat', 'silo-feat'], got: %v", features)
			}
		})

		t.Run("Scenario: user extensions are added to template customizations", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.user.toml", `[general]
user = "alice"
`)
				internal.WriteUserFile(t, siloUser, "devcontainer.user.json", `{"customizations": {"vscode": {"extensions": ["ms-python.python"]}}}`)
			})
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the .devcontainer.json should contain user's extensions
			data, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatalf("read .devcontainer.json: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("expected valid json: %v", err)
			}
			customizations := parsed["customizations"].(map[string]any)
			vscode := customizations["vscode"].(map[string]any)
			extensions := vscode["extensions"].([]any)
			found := false
			for _, e := range extensions {
				if e == "ms-python.python" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected user extension 'ms-python.python' in extensions, got: %v", extensions)
			}
		})
	})

	t.Run("Rule: Requires workspace to be initialized", func(t *testing.T) {
		t.Run("Scenario: devcontainer fails when workspace is not initialized", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			dir := t.TempDir()
			orig, _ := os.Getwd()
			t.Cleanup(func() { os.Chdir(orig) })
			os.Chdir(dir)

			// When I run `silo devcontainer`
			err := cmd.DevcontainerGenerate([]string{})

			// Then the exit code should not be 0
			if err == nil {
				t.Errorf("expected error, got nil")
			}
			// And the error should indicate ".silo/silo.toml" is missing
			if err != nil && !strings.Contains(err.Error(), ".silo/silo.toml") {
				t.Errorf("expected error to mention '.silo/silo.toml', got: %v", err)
			}
		})
	})

	t.Run("Rule: devcontainer is independent from workspace container", func(t *testing.T) {
		t.Run("Scenario: devcontainer command does not create the workspace container", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)
			internal.SetupUserConfig(t)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345": exec.Command("true"),
			})

			// When I run `silo devcontainer`
			cmd.DevcontainerGenerate([]string{})

			// Then no workspace container should be created
			mock.AssertNoExec("podman", "create", "<any>")
			mock.AssertNoExec("podman", "run", "<any>")
		})
	})

	t.Run("Rule: Creates .silo/devcontainer.json boilerplate for project-specific customization", func(t *testing.T) {
		t.Run("Scenario: .silo/devcontainer.json is created when not present", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)
			internal.SetupUserConfig(t)

			// Ensure .silo/devcontainer.json does not exist from prior tests
			os.RemoveAll(".silo/devcontainer.json")

			// When I run `silo devcontainer`
			output := internal.CaptureStdout(func() { cmd.DevcontainerGenerate([]string{}) })

			// Then a file ".silo/devcontainer.json" should be created
			if _, err := os.Stat(".silo/devcontainer.json"); os.IsNotExist(err) {
				t.Errorf(".silo/devcontainer.json was not created")
			}
			// And the output should contain "Creating .silo/devcontainer.json"
			if !strings.Contains(output, "Creating .silo/devcontainer.json") {
				t.Errorf("expected 'Creating .silo/devcontainer.json' in output, got: %s", output)
			}
		})

		t.Run("Scenario: .silo/devcontainer.json is skipped when already present", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)
			internal.SetupUserConfig(t)

			// Given a file ".silo/devcontainer.json" already exists
			if err := os.MkdirAll(".silo", 0755); err != nil {
				t.Fatal(err)
			}
			existing := []byte(`{"customizations": {"vscode": {"extensions": ["ms-python.python"]}}}`)
			if err := os.WriteFile(".silo/devcontainer.json", existing, 0644); err != nil {
				t.Fatal(err)
			}

			// When I run `silo devcontainer`
			output := internal.CaptureStdout(func() { cmd.DevcontainerGenerate([]string{}) })

			// Then the output should contain "'.silo/devcontainer.json' already exists"
			if !strings.Contains(output, "'.silo/devcontainer.json' already exists") {
				t.Errorf("expected 'already exists' in output, got: %s", output)
			}
			// And a file ".silo/devcontainer.json" should not be modified
			got, err := os.ReadFile(".silo/devcontainer.json")
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(existing) {
				t.Errorf("expected existing file to be preserved, got %s", string(got))
			}
		})

		t.Run("Scenario: .silo/devcontainer.json is created even when .devcontainer.json is skipped", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)
			internal.SetupUserConfig(t)

			// Given a file ".devcontainer.json" already exists
			if err := os.WriteFile(".devcontainer.json", []byte(`{"name": "custom"}`), 0644); err != nil {
				t.Fatal(err)
			}

			// When I run `silo devcontainer`
			cmd.DevcontainerGenerate([]string{})

			// Then a file ".silo/devcontainer.json" should be created
			if _, err := os.Stat(".silo/devcontainer.json"); os.IsNotExist(err) {
				t.Errorf(".silo/devcontainer.json was not created")
			}
		})
	})
}
