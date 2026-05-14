package features_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo devcontainer status — Show devcontainer status
// `silo devcontainer status` prints whether the devcontainer is running or stopped.
func TestFeatureDevcontainerStatus(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Reports devcontainer running state", func(t *testing.T) {
		t.Run("Scenario: status shows Running when devcontainer is up", func(t *testing.T) {
			internal.SetupMinimalWorkspace(t, "abc12345", "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
			})

			// When I run `silo devcontainer status`
			output := internal.CaptureStdout(func() { cmd.DevcontainerStatus() })

			// Then the output should contain "Running"
			if !strings.Contains(output, "Running") {
				t.Errorf("expected output to contain 'Running', got: %s", output)
			}
		})

		t.Run("Scenario: status shows Stopped when devcontainer is not running", func(t *testing.T) {
			internal.SetupMinimalWorkspace(t, "abc12345", "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "false"),
			})

			// When I run `silo devcontainer status`
			output := internal.CaptureStdout(func() { cmd.DevcontainerStatus() })

			// Then the output should contain "Stopped"
			if !strings.Contains(output, "Stopped") {
				t.Errorf("expected output to contain 'Stopped', got: %s", output)
			}
		})
	})

	t.Run("Rule: Requires workspace to be initialized", func(t *testing.T) {
		t.Run("Scenario: status fails when workspace is not initialized", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			internal.SetupEmptyWorkspace(t)

			// When I run `silo devcontainer status`
			err := cmd.DevcontainerStatus()

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
