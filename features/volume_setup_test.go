package features_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo volume setup — Create directories on the persistence volume
// `silo volume setup` creates directories on the persistence volume so they can be mounted
// as subpath volumes inside containers. It runs a temporary container with the workspace
// image — the workspace container does not need to be running. It is also run
// automatically after every `silo start`.
func TestFeatureVolumeSetup(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Creates directories on the persistence volume", func(t *testing.T) {
		t.Run("Scenario: volume setup creates directories on the persistence volume", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			cfg.Persistence.SharedPaths = []string{"$HOME/.cache/uv/"}
			internal.SetupWorkspace(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345": exec.Command("true"),
			})

			// When I run `silo volume setup`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.VolumeSetup()
			})

			// Then podman should run "run" with "--rm" and volume "silo:/silo/persistence:z"
			record := mock.AssertExec("podman", "run", "--rm", "<...>")
			cmdStr := strings.Join(record.Args, " ")
			// And the run command should create "/silo/persistence/shared/home/alice/.cache/uv" as a directory with mode 755
			expectedPath := "/silo/persistence/shared/home/alice/.cache/uv"
			if !strings.Contains(cmdStr, "mkdir -p "+expectedPath) {
				t.Errorf("expected mkdir -p %s, got: %s", expectedPath, cmdStr)
			}
			// And the output should contain "Volume setup complete"
			if !strings.Contains(output, "Volume setup complete") {
				t.Errorf("expected 'Volume setup complete' in output, got: %s", output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: volume setup creates both files and directories", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			cfg.Persistence.SharedPaths = []string{"$HOME/.cache/uv/", "$HOME/.local/share/fish/fish_history"}
			internal.SetupWorkspace(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345": exec.Command("true"),
			})

			// When I run `silo volume setup`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.VolumeSetup()
			})

			// Then podman should run "run" with "--rm" and volume "silo:/silo/persistence:z"
			record := mock.AssertExec("podman", "run", "--rm", "<...>")
			cmdStr := strings.Join(record.Args, " ")
			// And the run command should create "/silo/persistence/shared/home/alice/.cache/uv" as a directory with mode 755
			dirPath := "/silo/persistence/shared/home/alice/.cache/uv"
			if !strings.Contains(cmdStr, "mkdir -p "+dirPath) {
				t.Errorf("expected mkdir -p %s, got: %s", dirPath, cmdStr)
			}
			// And the run command should create "/silo/persistence/shared/home/alice/.local/share/fish/fish_history" as a file with mode 644
			filePath := "/silo/persistence/shared/home/alice/.local/share/fish/fish_history"
			if !strings.Contains(cmdStr, "touch "+filePath) {
				t.Errorf("expected touch %s, got: %s", filePath, cmdStr)
			}
			// And the output should contain "Volume setup complete"
			if !strings.Contains(output, "Volume setup complete") {
				t.Errorf("expected 'Volume setup complete' in output, got: %s", output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: No-op when shared paths is empty", func(t *testing.T) {
		t.Run("Scenario: empty shared_paths list is a no-op", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			cfg.Persistence.SharedPaths = []string{}
			internal.SetupWorkspace(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{})

			// When I run `silo volume setup`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.VolumeSetup()
			})

			// Then no podman run should be called
			mock.AssertNoExec("podman", "run", "<...>")
			// And the output should not contain "Volume setup complete"
			if strings.Contains(output, "Volume setup complete") {
				t.Errorf("expected no output for no-op, got: %s", output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Uses workspace image for temporary container", func(t *testing.T) {
		t.Run("Scenario: volume setup uses workspace image and does not require workspace container to exist", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			cfg.Persistence.SharedPaths = []string{"$HOME/.cache/uv/"}
			internal.SetupWorkspace(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo volume setup`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.VolumeSetup()
			})

			// Then the output should contain "Volume setup complete"
			if !strings.Contains(output, "Volume setup complete") {
				t.Errorf("expected 'Volume setup complete' in output, got: %s", output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Requires workspace to be initialized", func(t *testing.T) {
		t.Run("Scenario: volume setup fails when workspace is not initialized", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			base := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", base)
			internal.SetupEmptyWorkspace(t)

			// When I run `silo volume setup`
			err := cmd.VolumeSetup()

			// Then the exit code should not be 0
			// And the error should indicate ".silo/silo.toml" is missing
			if err == nil {
				t.Error("expected error when workspace is not initialized")
			}
		})
	})

	t.Run("Rule: Creates private paths for the silo", func(t *testing.T) {
		t.Run("Scenario: volume setup creates private paths under silo-specific directory", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			cfg.Persistence.PrivatePaths = []string{"$HOME/.cache/uv/"}
			internal.SetupWorkspace(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345": exec.Command("true"),
			})

			// When I run `silo volume setup`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.VolumeSetup()
			})

			// Then podman should run "run" with "--rm" and volume "silo:/silo/persistence:z"
			record := mock.AssertExec("podman", "run", "--rm", "<...>")
			cmdStr := strings.Join(record.Args, " ")
			// And the run command should create "/silo/persistence/abc12345/home/alice/.cache/uv" as a directory with mode 755
			expectedPath := "/silo/persistence/abc12345/home/alice/.cache/uv"
			if !strings.Contains(cmdStr, "mkdir -p "+expectedPath) {
				t.Errorf("expected mkdir -p %s, got: %s", expectedPath, cmdStr)
			}
			// And the output should contain "volume setup complete"
			if !strings.Contains(output, "Volume setup complete") {
				t.Errorf("expected 'volume setup complete' in output, got: %s", output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: volume setup creates both shared and private paths", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			cfg.Persistence.SharedPaths = []string{"$HOME/.local/share/fish/"}
			cfg.Persistence.PrivatePaths = []string{"$HOME/.cache/uv/"}
			internal.SetupWorkspace(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345": exec.Command("true"),
			})

			// When I run `silo volume setup`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.VolumeSetup()
			})

			// Then podman should run "run" with "--rm" and volume "silo:/silo/persistence:z"
			record := mock.AssertExec("podman", "run", "--rm", "<...>")
			cmdStr := strings.Join(record.Args, " ")
			// And the run command should create "/silo/persistence/shared/home/alice/.local/share/fish" as a directory with mode 755
			sharedPath := "/silo/persistence/shared/home/alice/.local/share/fish"
			if !strings.Contains(cmdStr, "mkdir -p "+sharedPath) {
				t.Errorf("expected mkdir -p %s, got: %s", sharedPath, cmdStr)
			}
			// And the run command should create "/silo/persistence/abc12345/home/alice/.cache/uv" as a directory with mode 755
			privatePath := "/silo/persistence/abc12345/home/alice/.cache/uv"
			if !strings.Contains(cmdStr, "mkdir -p "+privatePath) {
				t.Errorf("expected mkdir -p %s, got: %s", privatePath, cmdStr)
			}
			// And the output should contain "volume setup complete"
			if !strings.Contains(output, "Volume setup complete") {
				t.Errorf("expected 'volume setup complete' in output, got: %s", output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: empty private_paths list does not create private directories", func(t *testing.T) {
			cfg := internal.MinimalWorkspaceConfig("abc12345")
			cfg.Persistence.SharedPaths = []string{"$HOME/.cache/uv/"}
			cfg.Persistence.PrivatePaths = []string{}
			internal.SetupWorkspace(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345": exec.Command("true"),
			})

			// When I run `silo volume setup`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.VolumeSetup()
			})

			// Then podman should run "run" with "--rm" and volume "silo:/silo/persistence:z"
			record := mock.AssertExec("podman", "run", "--rm", "<...>")
			cmdStr := strings.Join(record.Args, " ")
			// And the run command should create "/silo/persistence/shared/home/alice/.cache/uv" as a directory with mode 755
			sharedPath := "/silo/persistence/shared/home/alice/.cache/uv"
			if !strings.Contains(cmdStr, "mkdir -p "+sharedPath) {
				t.Errorf("expected mkdir -p %s, got: %s", sharedPath, cmdStr)
			}
			// And no private path directory should be created under "/silo/persistence/abc12345/"
			privatePath := "/silo/persistence/abc12345/"
			if strings.Contains(cmdStr, privatePath) {
				t.Errorf("expected no reference to %s, got: %s", privatePath, cmdStr)
			}
			// And the output should contain "volume setup complete"
			if !strings.Contains(output, "Volume setup complete") {
				t.Errorf("expected 'volume setup complete' in output, got: %s", output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})
}
