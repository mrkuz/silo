package features_test

import (
	"os"
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

	t.Run("Rule: Stops the running container", func(t *testing.T) {
		t.Run("Scenario: stop terminates the container immediately", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman stop -t 0 silo-abc12345":                                     exec.Command("true"),
			})

			// When I run `silo stop`
			output := internal.CaptureStdout(func() { cmd.Stop() })

			// Then podman should run "stop" with "-t" and "0" on "silo-abc12345"
			mock.AssertExec("podman", "stop", "-t", "0", "silo-abc12345")
			// And the output should contain "Stopping silo-abc12345..."
			if !strings.Contains(output, "Stopping silo-abc12345...") {
				t.Errorf("expected output to contain 'Stopping silo-abc12345...', got: %s", output)
			}
		})
	})

	t.Run("Rule: No-op if container is already stopped", func(t *testing.T) {
		t.Run("Scenario: stopped container is not an error", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
			})

			// When I run `silo stop`
			output := internal.CaptureStdout(func() { cmd.Stop() })

			// Then the output should contain "silo-abc12345 is not running"
			if !strings.Contains(output, "silo-abc12345 is not running") {
				t.Errorf("expected output to contain 'silo-abc12345 is not running', got: %s", output)
			}
			// And no podman stop should be called
			mock.AssertNoExec("podman", "stop", "<any>")
		})
	})

	t.Run("Rule: Stop does not remove container or image", func(t *testing.T) {
		t.Run("Scenario: stop only stops, it does not remove anything", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman stop <any>": exec.Command("true"),
			})

			// When I run `silo stop`
			cmd.Stop()

			// Then podman should run "stop" on "silo-abc12345"
			// But podman should not run "rm" on "silo-abc12345"
			mock.AssertNoExec("podman", "rm", "<any>")
			// And podman should not run "rmi" on "silo-abc12345"
			mock.AssertNoExec("podman", "rmi", "<any>")
		})
	})

	t.Run("Rule: Requires workspace to be initialized", func(t *testing.T) {
		t.Run("Scenario: stop fails when workspace is not initialized", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			dir := t.TempDir()
			orig, _ := os.Getwd()
			t.Cleanup(func() { os.Chdir(orig) })
			os.Chdir(dir)

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
