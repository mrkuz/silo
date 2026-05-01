package features_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo devcontainer stop — Stop the devcontainer
// `silo devcontainer stop` stops the devcontainer. It is a no-op if the
// devcontainer is not running.
func TestFeatureDevcontainerStop(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Stops the running devcontainer", func(t *testing.T) {
		t.Run("Scenario: stop stops the devcontainer", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
				"podman stop -t 0 silo-abc12345-dev":                                     exec.Command("true"),
			})

			// When I run `silo devcontainer stop`
			err := cmd.DevcontainerStop()

			// Then podman should run "stop" with "-t" and "0" on "silo-abc12345-dev"
			mock.AssertExec("podman", "stop", "-t", "0", "silo-abc12345-dev")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: stop prints a message", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
				"podman stop -t 0 silo-abc12345-dev":                                     exec.Command("true"),
			})

			// When I run `silo devcontainer stop`
			output := internal.CaptureStdout(func() { cmd.DevcontainerStop() })

			// Then the output should contain "Stopping silo-abc12345-dev..."
			if !strings.Contains(output, "Stopping silo-abc12345-dev...") {
				t.Errorf("expected output to contain 'Stopping silo-abc12345-dev...', got: %s", output)
			}
		})

		t.Run("Scenario: stop does not stop the workspace container", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345":     exec.Command("echo", "true"),
				"podman stop <any>": exec.Command("true"),
			})

			// When I run `silo devcontainer stop`
			cmd.DevcontainerStop()

			// Then podman should run "stop" on "silo-abc12345-dev"
			// But podman should not run "stop" on "silo-abc12345"
			mock.AssertNoExec("podman", "stop", "silo-abc12345")
		})
	})

	t.Run("Rule: No-op if devcontainer is not running", func(t *testing.T) {
		t.Run("Scenario: stopped devcontainer is not an error", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "false"),
			})

			// When I run `silo devcontainer stop`
			output := internal.CaptureStdout(func() { cmd.DevcontainerStop() })

			// Then the output should contain "silo-abc12345-dev is not running"
			if !strings.Contains(output, "silo-abc12345-dev is not running") {
				t.Errorf("expected output to contain 'silo-abc12345-dev is not running', got: %s", output)
			}
			// And no podman stop should be called
			mock.AssertNoExec("podman", "stop", "<any>")
		})
	})

	t.Run("Rule: Requires workspace to be initialized", func(t *testing.T) {
		t.Run("Scenario: devcontainer stop fails when workspace is not initialized", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			dir := t.TempDir()
			orig, _ := os.Getwd()
			t.Cleanup(func() { os.Chdir(orig) })
			os.Chdir(dir)

			// When I run `silo devcontainer stop`
			err := cmd.DevcontainerStop()

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
}
