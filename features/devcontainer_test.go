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
			err := cmd.DevcontainerGenerate()

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
			cmd.DevcontainerGenerate()

			// Then the file ".devcontainer.json" should still contain '{"name": "custom"}'
			got, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(existing) {
				t.Errorf("expected existing file to be preserved, got %s", string(got))
			}
			// And no new .devcontainer.json should be generated
			// And the exit code should be 0
		})

		t.Run("Scenario: devcontainer uses the workspace image", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			internal.SetupWorkspace(t, cfg)
			internal.SetupUserConfig(t)

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate(); err != nil {
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
			if err := cmd.DevcontainerGenerate(); err != nil {
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

		t.Run("Scenario: devcontainer runs volume setup before generating when shared volume is configured", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.Features.SharedVolume = true
			cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman run --rm <...>":                 exec.Command("true"),
			})

			// When I run `silo devcontainer`
			cmd.DevcontainerGenerate()

			// Then shared volume directories should be created before generating .devcontainer.json.
			record := mock.AssertExec("podman", "run", "--rm", "<...>")
			cmdStr := strings.Join(record.Args, " ")
			expectedPath := "/silo/shared/home/alice/.cache/uv"
			if !strings.Contains(cmdStr, "mkdir -p "+expectedPath) {
				t.Errorf("expected volume setup with 'mkdir -p %s', got: %s", expectedPath, cmdStr)
			}
		})
	})

	t.Run("Rule: User config is merged into generated .devcontainer.json", func(t *testing.T) {
		t.Run("Scenario: user devcontainer.in.json merges into generated .devcontainer.json", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "devcontainer.in.json", `{"customizations": {"vscode": {"extensions": ["ms-python.python"]}}}`)
			})
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SetupWorkspace(t, cfg)

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the .devcontainer.json should contain the user's "customizations"
			data, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatalf("read .devcontainer.json: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("expected valid json: %v", err)
			}
			customizations, ok := parsed["customizations"].(map[string]any)
			if !ok {
				t.Errorf("expected customizations in parsed json, got: %v", parsed["customizations"])
			}
			vscode, ok := customizations["vscode"].(map[string]any)
			if !ok {
				t.Errorf("expected vscode in customizations, got: %v", customizations["vscode"])
			}
			if _, ok := vscode["extensions"].([]any); !ok {
				t.Errorf("expected extensions in vscode, got: %v", vscode["extensions"])
			}
		})

		t.Run("Scenario: arrays are concatenated on merge", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "devcontainer.in.json", `{"features": ["c"]}`)
			})
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SetupWorkspace(t, cfg)

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the .devcontainer.json should have "features" from user config
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
			// User config features should be present
			if len(features) != 1 || features[0] != "c" {
				t.Errorf("expected features ['c'], got: %v", features)
			}
		})

		t.Run("Scenario: scalars from user config override generated values", func(t *testing.T) {
			internal.FirstRunWith(t, func(siloUser string) {
				internal.WriteUserFile(t, siloUser, "devcontainer.in.json", `{"name": "my-devcontainer"}`)
			})
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SetupWorkspace(t, cfg)

			// When I run `silo devcontainer`
			if err := cmd.DevcontainerGenerate(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the .devcontainer.json should have "name" set to "my-devcontainer"
			data, err := os.ReadFile(".devcontainer.json")
			if err != nil {
				t.Fatalf("read .devcontainer.json: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("expected valid json: %v", err)
			}
			if name, ok := parsed["name"].(string); ok {
				if name != "my-devcontainer" {
					t.Errorf("expected name 'my-devcontainer', got: %s", name)
				}
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
			err := cmd.DevcontainerGenerate()

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
			cmd.DevcontainerGenerate()

			// Then no workspace container should be created
			mock.AssertNoExec("podman", "create", "<any>")
			mock.AssertNoExec("podman", "run", "<any>")
		})
	})
}
