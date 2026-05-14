package features_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo devcontainer stop — Stop and remove the devcontainer
// `silo devcontainer stop` stops the devcontainer immediately (no grace period),
// then removes it. If the devcontainer is not running, it prints a message and still
// attempts to remove. If the devcontainer does not exist, it prints "not found".
func TestFeatureDevcontainerStop(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Running devcontainer is stopped and removed", func(t *testing.T) {
		t.Run("Scenario: stop terminates and removes the devcontainer", func(t *testing.T) {
			internal.SetupMinimalWorkspace(t, "abc12345", "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
				"podman stop -t 0 silo-abc12345-dev":                                     exec.Command("true"),
				"podman rm -f silo-abc12345-dev":                                        exec.Command("true"),
			})

			// When I run `silo devcontainer stop`
			output := internal.CaptureStdout(func() { cmd.DevcontainerStop() })

			// Then podman should run "stop" with "-t" and "0" on "silo-abc12345-dev"
			mock.AssertExec("podman", "stop", "-t", "0", "silo-abc12345-dev")
			// And podman should run "rm" with "-f" on "silo-abc12345-dev"
			mock.AssertExec("podman", "rm", "-f", "silo-abc12345-dev")
			// And the output should contain "Stopping silo-abc12345-dev..."
			if !strings.Contains(output, "Stopping silo-abc12345-dev...") {
				t.Errorf("expected output to contain 'Stopping silo-abc12345-dev...', got: %s", output)
			}
			// And the output should contain "Removing silo-abc12345-dev..."
			if !strings.Contains(output, "Removing silo-abc12345-dev...") {
				t.Errorf("expected output to contain 'Removing silo-abc12345-dev...', got: %s", output)
			}
			// And the exit code should be 0
		})

		t.Run("Scenario: stop does not affect the workspace container", func(t *testing.T) {
			internal.SetupMinimalWorkspace(t, "abc12345", "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345":     exec.Command("echo", "true"),
				"podman stop -t 0 silo-abc12345-dev":                                     exec.Command("true"),
				"podman rm -f silo-abc12345-dev":                                        exec.Command("true"),
			})

			// When I run `silo devcontainer stop`
			cmd.DevcontainerStop()

			// Then podman should run "stop" on "silo-abc12345-dev"
			mock.AssertExec("podman", "stop", "-t", "0", "silo-abc12345-dev")
			// And podman should run "rm" on "silo-abc12345-dev"
			mock.AssertExec("podman", "rm", "-f", "silo-abc12345-dev")
			// But podman should not run "stop" on "silo-abc12345"
			mock.AssertNoExec("podman", "stop", "silo-abc12345")
			// And podman should not run "rm" on "silo-abc12345"
			mock.AssertNoExec("podman", "rm", "silo-abc12345")
		})
	})

	t.Run("Rule: Stopped devcontainer is removed", func(t *testing.T) {
		t.Run("Scenario: stopped devcontainer prints message and is removed", func(t *testing.T) {
			internal.SetupMinimalWorkspace(t, "abc12345", "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "false"),
				"podman rm -f silo-abc12345-dev":                                                  exec.Command("true"),
			})

			// When I run `silo devcontainer stop`
			output := internal.CaptureStdout(func() { cmd.DevcontainerStop() })

			// Then podman should run "rm" with "-f" on "silo-abc12345-dev"
			mock.AssertExec("podman", "rm", "-f", "silo-abc12345-dev")
			// And the output should contain "silo-abc12345-dev is not running"
			if !strings.Contains(output, "silo-abc12345-dev is not running") {
				t.Errorf("expected output to contain 'silo-abc12345-dev is not running', got: %s", output)
			}
			// And the output should contain "Removing silo-abc12345-dev..."
			if !strings.Contains(output, "Removing silo-abc12345-dev...") {
				t.Errorf("expected output to contain 'Removing silo-abc12345-dev...', got: %s", output)
			}
			// And the exit code should be 0
		})
	})

	t.Run("Rule: Non-existing devcontainer prints not found", func(t *testing.T) {
		t.Run("Scenario: absent devcontainer prints not found and exits 0", func(t *testing.T) {
			internal.SetupMinimalWorkspace(t, "abc12345", "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345-dev": exec.Command("false"),
			})

			// When I run `silo devcontainer stop`
			output := internal.CaptureStdout(func() { cmd.DevcontainerStop() })

			// Then the output should contain "silo-abc12345-dev not found"
			if !strings.Contains(output, "silo-abc12345-dev not found") {
				t.Errorf("expected output to contain 'silo-abc12345-dev not found', got: %s", output)
			}
		})
	})

	t.Run("Rule: Requires workspace to be initialized", func(t *testing.T) {
		t.Run("Scenario: devcontainer stop fails when workspace is not initialized", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			internal.SetupEmptyWorkspace(t)

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
