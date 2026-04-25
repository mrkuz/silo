package features_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo status — Show workspace container status
// `silo status` prints whether the workspace container is currently running or stopped.
// It requires the workspace to have been initialized first.
func TestFeatureStatus(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Reports container running state", func(t *testing.T) {
		t.Run("Scenario: status shows Running when container is up", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
			})

			// When I run `silo status`
			output := internal.CaptureStdout(func() { cmd.Status() })

			// Then the output should contain "Running"
			if !strings.Contains(output, "Running") {
				t.Errorf("expected output to contain 'Running', got: %s", output)
			}
		})

		t.Run("Scenario: status shows Stopped when container is not running", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
			})

			// When I run `silo status`
			output := internal.CaptureStdout(func() { cmd.Status() })

			// Then the output should contain "Stopped"
			if !strings.Contains(output, "Stopped") {
				t.Errorf("expected output to contain 'Stopped', got: %s", output)
			}
		})
	})

	t.Run("Rule: Requires workspace to be initialized", func(t *testing.T) {
		t.Run("Scenario: status fails when workspace is not initialized", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			dir := t.TempDir()
			orig, _ := os.Getwd()
			t.Cleanup(func() { os.Chdir(orig) })
			os.Chdir(dir)

			// When I run `silo status`
			err := cmd.Status()

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