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
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			internal.SetupWorkspaceFiles(t, cfg)
			internal.SetupUserFiles(t)

			// When I run `silo devcontainer`
			output := internal.CaptureStdout(func() { cmd.DevcontainerGenerate([]string{}) })

			// Then a file ".devcontainer.json" should be created
			if _, statErr := os.Stat(".devcontainer.json"); os.IsNotExist(statErr) {
				t.Errorf(".devcontainer.json was not created")
			}
			// And the output should contain "Creating .devcontainer.json..."
			if !strings.Contains(output, "Creating .devcontainer.json...") {
				t.Errorf("expected 'Creating .devcontainer.json...' in output, got: %s", output)
			}
		})

		t.Run("Scenario: existing .devcontainer.json is not overwritten", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			internal.SetupWorkspaceFiles(t, cfg)
			internal.SetupUserFiles(t)

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
			if !strings.Contains(output, ".devcontainer.json already exists") {
				t.Errorf("expected 'already exists' in output, got: %s", output)
			}
		})

		t.Run("Scenario: --update overwrites existing .devcontainer.json", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			internal.SetupWorkspaceFiles(t, cfg)
			internal.SetupUserFiles(t)

			// Given a file ".devcontainer.json" already exists with content '{"name": "custom"}'
			existing := []byte(`{"name": "custom"}`)
			if err := os.WriteFile(".devcontainer.json", existing, 0644); err != nil {
				t.Fatal(err)
			}

			// When I run `silo devcontainer --update`
			output := internal.CaptureStdout(func() { cmd.DevcontainerGenerate([]string{"--update"}) })

			// Then the output should contain ".devcontainer.json updated"
			if !strings.Contains(output, ".devcontainer.json updated") {
				t.Errorf("expected '.devcontainer.json updated' in output, got: %s", output)
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

		t.Run("Scenario: --update does not affect .silo/devcontainer.json handling", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			internal.SetupWorkspaceFiles(t, cfg)
			internal.SetupUserFiles(t)

			// Given a file ".silo/devcontainer.json" already exists with content '{"custom": true}'
			if err := os.MkdirAll(".silo", 0755); err != nil {
				t.Fatal(err)
			}
			existing := []byte(`{"custom": true}`)
			if err := os.WriteFile(".silo/devcontainer.json", existing, 0644); err != nil {
				t.Fatal(err)
			}

			// When I run `silo devcontainer --update`
			output := internal.CaptureStdout(func() { cmd.DevcontainerGenerate([]string{"--update"}) })

			// Then the output should contain ".silo/devcontainer.json already exists"
			if !strings.Contains(output, ".silo/devcontainer.json already exists") {
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
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			internal.SetupWorkspaceFiles(t, cfg)
			internal.SetupUserFiles(t)

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
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			internal.SetupWorkspaceFiles(t, cfg)
			internal.SetupUserFiles(t)

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
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			internal.SetupWorkspaceFiles(t, cfg)
			internal.SetupUserFiles(t)

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
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			cfg.Persistence.SharedPaths = []string{"$HOME/.cache/uv/"}
			internal.SetupWorkspace(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman run --rm <...>":                 exec.Command("true"),
			})

			// When I run `silo devcontainer`
			cmd.DevcontainerGenerate([]string{})

			// Then shared volume directories should be created before generating .devcontainer.json.
			record := mock.AssertExec("podman", "run", "--rm", "<...>")
			cmdStr := strings.Join(record.Args, " ")
			expectedPath := "/silo/persistence/shared/home/alice/.cache/uv"
			if !strings.Contains(cmdStr, "mkdir -p "+expectedPath) {
				t.Errorf("expected volume setup with 'mkdir -p %s', got: %s", expectedPath, cmdStr)
			}
		})

		t.Run("Scenario: devcontainer includes forwardPorts when network ports are configured", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			cfg.Network.Ports = []string{"8080:8080", "3000:3000"}
			internal.SetupWorkspaceFiles(t, cfg)
			internal.SetupUserFiles(t, "alice")

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the .devcontainer.json should have "forwardPorts" with all elements "8080:8080", "3000:3000" in order
			data, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatalf("read .devcontainer.json: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("expected valid json: %v", err)
			}
			forwardPorts, ok := parsed["forwardPorts"].([]any)
			if !ok {
				t.Fatalf("expected forwardPorts to be array, got: %v", parsed["forwardPorts"])
			}
			if len(forwardPorts) != 2 {
				t.Errorf("expected 2 forwardPorts, got %d: %v", len(forwardPorts), forwardPorts)
			}
			if forwardPorts[0] != "8080:8080" || forwardPorts[1] != "3000:3000" {
				t.Errorf("expected ['8080:8080', '3000:3000'], got: %v", forwardPorts)
			}
		})

		t.Run("Scenario: devcontainer omits forwardPorts when no network ports are configured", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			cfg.Network.Ports = []string{}
			internal.SetupWorkspaceFiles(t, cfg)
			internal.SetupUserFiles(t, "alice")

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the .devcontainer.json should not have "forwardPorts"
			data, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatalf("read .devcontainer.json: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("expected valid json: %v", err)
			}
			if _, ok := parsed["forwardPorts"]; ok {
				t.Errorf("expected no forwardPorts in .devcontainer.json, got: %v", parsed["forwardPorts"])
			}
		})

		t.Run("Scenario: devcontainer includes limits in runArgs", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			cfg.Limits.CPUs = 2
			cfg.Limits.Memory = 4096
			cfg.Limits.Processes = 1024
			internal.SetupWorkspaceFiles(t, cfg)
			internal.SetupUserFiles(t, "alice")

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the .devcontainer.json should have limits in runArgs
			data, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatalf("read .devcontainer.json: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("expected valid json: %v", err)
			}
			runArgs, ok := parsed["runArgs"].([]any)
			if !ok {
				t.Fatalf("expected runArgs to be array, got: %v", parsed["runArgs"])
			}
			runArgsStr := toStringSlice(runArgs)
			joined := strings.Join(runArgsStr, " ")
			if !strings.Contains(joined, "--cpus=2") {
				t.Errorf("expected --cpus=2 in runArgs, got: %v", runArgs)
			}
			if !strings.Contains(joined, "--memory=4096m") {
				t.Errorf("expected --memory=4096m in runArgs, got: %v", runArgs)
			}
			if !strings.Contains(joined, "--pids-limit=1024") {
				t.Errorf("expected --pids-limit=1024 in runArgs, got: %v", runArgs)
			}
		})

		t.Run("Scenario: devcontainer includes unlimited values for zero or negative limits", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			cfg.Limits.CPUs = 0
			cfg.Limits.Memory = -1
			cfg.Limits.Processes = 0
			internal.SetupWorkspaceFiles(t, cfg)
			internal.SetupUserFiles(t, "alice")

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the .devcontainer.json should have unlimited limit flags in runArgs
			data, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatalf("read .devcontainer.json: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("expected valid json: %v", err)
			}
			runArgs, ok := parsed["runArgs"].([]any)
			if !ok {
				t.Fatalf("expected runArgs to be array, got: %v", parsed["runArgs"])
			}
			runArgsStr := toStringSlice(runArgs)
			joined := strings.Join(runArgsStr, " ")
			if !strings.Contains(joined, "--cpus=0") {
				t.Errorf("expected --cpus=0 in runArgs, got: %v", runArgs)
			}
			if !strings.Contains(joined, "--memory=0") {
				t.Errorf("expected --memory=0 in runArgs, got: %v", runArgs)
			}
			if !strings.Contains(joined, "--pids-limit=-1") {
				t.Errorf("expected --pids-limit=-1 in runArgs, got: %v", runArgs)
			}
		})
	})

	t.Run("Rule: Merge order: template wins > .silo > user", func(t *testing.T) {
		t.Run("Scenario: template values override user config", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.user.toml", `
				[general]
				user = "alice"
				`)
				internal.WriteUserFile(t, siloUser, "devcontainer.user.json", `{"name": "my-devcontainer"}`)
			})
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			internal.SetupWorkspaceFiles(t, cfg)

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
				internal.WriteUserFile(t, siloUser, "silo.user.toml", `
				[general]
				user = "alice"
				`)
				internal.WriteUserFile(t, siloUser, "devcontainer.user.json", `{"name": "user-name", "custom": "user-value"}`)
			})
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			internal.SetupWorkspaceFiles(t, cfg)

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

		t.Run("Scenario: arrays from all sources are concatenated in order", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.user.toml", `
				[general]
				user = "alice"
				`)
				internal.WriteUserFile(t, siloUser, "devcontainer.user.json", `{"forwardPorts": ["3000:3000"]}`)
			})
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			internal.SetupWorkspaceFiles(t, cfg)

			// Given the workspace has ".silo/devcontainer.json" with content '{"forwardPorts": ["8080:8080"]}'
			if err := os.MkdirAll(".silo", 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(".silo/devcontainer.json", []byte(`{"forwardPorts": ["8080:8080"]}`), 0644); err != nil {
				t.Fatal(err)
			}

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the .devcontainer.json should have "forwardPorts" with all elements "3000:3000", "8080:8080" in order
			data, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatalf("read .devcontainer.json: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("expected valid json: %v", err)
			}
			forwardPorts, ok := parsed["forwardPorts"].([]any)
			if !ok {
				t.Fatalf("expected forwardPorts to be array, got: %v", parsed["forwardPorts"])
			}
			if len(forwardPorts) != 2 {
				t.Errorf("expected 2 forwardPorts, got %d: %v", len(forwardPorts), forwardPorts)
			}
			if forwardPorts[0] != "3000:3000" || forwardPorts[1] != "8080:8080" {
				t.Errorf("expected ['3000:3000', '8080:8080'], got: %v", forwardPorts)
			}
		})

		t.Run("Scenario: user extensions are added to template customizations", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "silo.user.toml", `
				[general]
				user = "alice"
				`)
				internal.WriteUserFile(t, siloUser, "devcontainer.user.json", `{"customizations": {"vscode": {"extensions": ["ms-python.python"]}}}`)
			})
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			internal.SetupWorkspaceFiles(t, cfg)

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
			internal.SetupEmptyWorkspace(t)

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

	t.Run("Rule: Creates .silo/devcontainer.json boilerplate for project-specific customization", func(t *testing.T) {
		t.Run("Scenario: .silo/devcontainer.json is created when not present", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			internal.SetupWorkspaceFiles(t, cfg)
			internal.SetupUserFiles(t)

			// Ensure .silo/devcontainer.json does not exist from prior tests
			os.RemoveAll(".silo/devcontainer.json")

			// When I run `silo devcontainer`
			output := internal.CaptureStdout(func() { cmd.DevcontainerGenerate([]string{}) })

			// Then a file ".silo/devcontainer.json" should be created
			if _, err := os.Stat(".silo/devcontainer.json"); os.IsNotExist(err) {
				t.Errorf(".silo/devcontainer.json was not created")
			}
			// And the output should contain "Creating .silo/devcontainer.json..."
			if !strings.Contains(output, "Creating .silo/devcontainer.json...") {
				t.Errorf("expected 'Creating .silo/devcontainer.json...' in output, got: %s", output)
			}
		})

		t.Run("Scenario: .silo/devcontainer.json is skipped when already present", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			internal.SetupWorkspaceFiles(t, cfg)
			internal.SetupUserFiles(t)

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

			// Then the output should contain ".silo/devcontainer.json already exists"
			if !strings.Contains(output, ".silo/devcontainer.json already exists") {
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
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			internal.SetupWorkspaceFiles(t, cfg)
			internal.SetupUserFiles(t)

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

func toStringSlice(in []any) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = v.(string)
	}
	return out
}
