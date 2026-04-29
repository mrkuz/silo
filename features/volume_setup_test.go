package features_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo volume setup — Create directories on the shared volume
// `silo volume setup` creates directories on the shared volume so they can be mounted
// as subpath volumes inside containers. It runs a temporary container with the user
// image — the workspace container does not need to be running. It is also run
// automatically after every `silo start`.
func TestFeatureVolumeSetup(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Creates directories on the shared volume", func(t *testing.T) {
		t.Run("Scenario: volume setup creates directories on the shared volume", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.Features.SharedVolume = true
			cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice": exec.Command("true"),
			})

			// When I run `silo volume setup`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.VolumeSetup()
			})

			// Then podman should run "run" with "--rm" and volume "silo-shared:/silo/shared:z"
			record := mock.AssertExec("podman", "run", "--rm", "<...>")
			cmdStr := strings.Join(record.Args, " ")
			// And the run command should create "/silo/shared/home/alice/.cache/uv" as a directory with mode 755
			expectedPath := "/silo/shared/home/alice/.cache/uv"
			if !strings.Contains(cmdStr, "mkdir -p "+expectedPath) {
				t.Errorf("expected mkdir -p %s, got: %s", expectedPath, cmdStr)
			}
			// And the output should contain "volume setup complete"
			if !strings.Contains(output, "volume setup complete") {
				t.Errorf("expected 'volume setup complete' in output, got: %s", output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: volume setup creates both files and directories", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.Features.SharedVolume = true
			cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/", "$HOME/.local/share/fish/fish_history"}
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice": exec.Command("true"),
			})

			// When I run `silo volume setup`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.VolumeSetup()
			})

			// Then podman should run "run" with "--rm" and volume "silo-shared:/silo/shared:z"
			record := mock.AssertExec("podman", "run", "--rm", "<...>")
			cmdStr := strings.Join(record.Args, " ")
			// And the run command should create "/silo/shared/home/alice/.cache/uv" as a directory with mode 755
			dirPath := "/silo/shared/home/alice/.cache/uv"
			if !strings.Contains(cmdStr, "mkdir -p "+dirPath) {
				t.Errorf("expected mkdir -p %s, got: %s", dirPath, cmdStr)
			}
			// And the run command should create "/silo/shared/home/alice/.local/share/fish/fish_history" as a file with mode 644
			filePath := "/silo/shared/home/alice/.local/share/fish/fish_history"
			if !strings.Contains(cmdStr, "touch "+filePath) {
				t.Errorf("expected touch %s, got: %s", filePath, cmdStr)
			}
			// And the output should contain "volume setup complete"
			if !strings.Contains(output, "volume setup complete") {
				t.Errorf("expected 'volume setup complete' in output, got: %s", output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: No-op when shared volume is not configured", func(t *testing.T) {
		t.Run("Scenario: disabled shared volume is a no-op", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.Features.SharedVolume = false
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{})

			// When I run `silo volume setup`
			err := cmd.VolumeSetup()

			// Then no podman run should be called
			mock.AssertNoExec("podman", "run", "<...>")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: empty paths list is a no-op", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.Features.SharedVolume = true
			cfg.SharedVolume.Paths = []string{}
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{})

			// When I run `silo volume setup`
			err := cmd.VolumeSetup()

			// Then no podman run should be called
			mock.AssertNoExec("podman", "run", "<...>")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Uses a temporary container, not the workspace container", func(t *testing.T) {
		t.Run("Scenario: volume setup does not require workspace container to exist", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.Features.SharedVolume = true
			cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo volume setup`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.VolumeSetup()
			})

			// Then the output should contain "volume setup complete"
			if !strings.Contains(output, "volume setup complete") {
				t.Errorf("expected 'volume setup complete' in output, got: %s", output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Builds user image if missing before running temporary container", func(t *testing.T) {
		t.Run("Scenario: missing user image triggers build", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.Features.SharedVolume = true
			cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice": exec.Command("false"),
				"podman build <...>":             exec.Command("true"),
			})

			// When I run `silo volume setup`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.VolumeSetup()
			})

			// Then the user image "silo-alice" should be built
			mock.AssertExec("podman", "build", "<...>")
			// And directories should be created on the shared volume
			mock.AssertExec("podman", "run", "--rm", "<...>")
			// And the output should contain "volume setup complete"
			if !strings.Contains(output, "volume setup complete") {
				t.Errorf("expected 'volume setup complete' in output, got: %s", output)
			}
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
			dir := t.TempDir()
			orig, _ := os.Getwd()
			t.Cleanup(func() { os.Chdir(orig) })
			os.Chdir(dir)

			// When I run `silo volume setup`
			err := cmd.VolumeSetup()

			// Then the exit code should not be 0
			// And the error should indicate ".silo/silo.toml" is missing
			if err == nil {
				t.Error("expected error when workspace is not initialized")
			}
		})
	})
}
