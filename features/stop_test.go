package features_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo stop — Stop the workspace container
// `silo stop` stops the running workspace container immediately (no grace period).
// It is a no-op if the container is already stopped.
func TestFeatureStop(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Running container is stopped and removed", func(t *testing.T) {
		t.Run("Scenario: stop terminates and removes the container", func(t *testing.T) {
			internal.SetupMinimalWorkspace(t, "abc12345", "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman stop -t 0 silo-abc12345":                                     exec.Command("true"),
				"podman rm -f silo-abc12345":                                         exec.Command("true"),
			})

			// When I run `silo stop`
			output := internal.CaptureStdout(func() { cmd.Stop() })

			// Then podman should run "stop" with "-t" and "0" on "silo-abc12345"
			mock.AssertExec("podman", "stop", "-t", "0", "silo-abc12345")
			// And podman should run "rm" with "-f" on "silo-abc12345"
			mock.AssertExec("podman", "rm", "-f", "silo-abc12345")
			// And the output should contain "Stopping silo-abc12345..."
			if !strings.Contains(output, "Stopping silo-abc12345...") {
				t.Errorf("expected output to contain 'Stopping silo-abc12345...', got: %s", output)
			}
			// And the output should contain "Removing silo-abc12345..."
			if !strings.Contains(output, "Removing silo-abc12345...") {
				t.Errorf("expected output to contain 'Removing silo-abc12345...', got: %s", output)
			}
			// And the exit code should be 0
		})
	})

	t.Run("Rule: Stopped container is removed", func(t *testing.T) {
		t.Run("Scenario: stopped container prints message and is removed", func(t *testing.T) {
			internal.SetupMinimalWorkspace(t, "abc12345", "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
				"podman rm -f silo-abc12345":                                         exec.Command("true"),
			})

			// When I run `silo stop`
			output := internal.CaptureStdout(func() { cmd.Stop() })

			// Then podman should run "rm" with "-f" on "silo-abc12345"
			mock.AssertExec("podman", "rm", "-f", "silo-abc12345")
			// And the output should contain "silo-abc12345 is not running"
			if !strings.Contains(output, "silo-abc12345 is not running") {
				t.Errorf("expected output to contain 'silo-abc12345 is not running', got: %s", output)
			}
			// And the output should contain "Removing silo-abc12345..."
			if !strings.Contains(output, "Removing silo-abc12345...") {
				t.Errorf("expected output to contain 'Removing silo-abc12345...', got: %s", output)
			}
			// And the exit code should be 0
		})

		t.Run("Scenario: absent container prints not found and exits 0", func(t *testing.T) {
			internal.SetupMinimalWorkspace(t, "abc12345", "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo stop`
			output := internal.CaptureStdout(func() { cmd.Stop() })

			// Then the output should contain "silo-abc12345 not found"
			if !strings.Contains(output, "silo-abc12345 not found") {
				t.Errorf("expected output to contain 'silo-abc12345 not found', got: %s", output)
			}
		})
	})

	t.Run("Rule: Requires workspace to be initialized", func(t *testing.T) {
		t.Run("Scenario: stop fails when workspace is not initialized", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			internal.SetupEmptyWorkspace(t)

			// When I run `silo stop`
			err := cmd.Stop()

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
