package features_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo rm — Remove the workspace container
// `silo rm` removes the workspace container. It refuses to remove a running
// container unless `--force` is given. It does not remove the image.
func TestFeatureRm(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Removes a stopped container", func(t *testing.T) {
		t.Run("Scenario: stopped container is removed without --force", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman rm -f silo-abc12345":                                        exec.Command("true"),
			})

			// When I run `silo rm`
			output := internal.CaptureStdout(func() { cmd.Remove(nil) })

			// Then podman should run "rm" with "-f" on "silo-abc12345"
			mock.AssertExec("podman", "rm", "-f", "silo-abc12345")
			// And the output should contain "Removing silo-abc12345..."
			if !strings.Contains(output, "Removing silo-abc12345...") {
				t.Errorf("expected output to contain 'Removing silo-abc12345...', got: %s", output)
			}
		})
	})

	t.Run("Rule: Refuses to remove a running container without --force", func(t *testing.T) {
		t.Run("Scenario: running container without --force returns an error", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman container exists silo-abc12345":                              exec.Command("true"),
			})

			// When I run `silo rm`
			err := cmd.Remove(nil)

			// Then the exit code should not be 0
			if err == nil {
				t.Errorf("expected error, got nil")
			}
			// And the error should indicate "silo-abc12345 is running"
			if err != nil && !strings.Contains(err.Error(), "silo-abc12345 is running") {
				t.Errorf("expected error to mention 'silo-abc12345 is running', got: %v", err)
			}
			// And the container "silo-abc12345" should still exist
			mock.AssertExec("podman", "container", "exists", "silo-abc12345")
		})
	})

	t.Run("Rule: --force stops and removes a running container", func(t *testing.T) {
		t.Run("Scenario: --force stops running container before removing", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman stop -t 0 silo-abc12345":                                   exec.Command("true"),
				"podman rm -f silo-abc12345":                                        exec.Command("true"),
			})

			// When I run `silo rm --force`
			err := cmd.Remove([]string{"--force"})

			// Then podman should run "stop" with "-t" and "0" on "silo-abc12345"
			mock.AssertExec("podman", "stop", "-t", "0", "silo-abc12345")
			// And podman should run "rm" with "-f" on "silo-abc12345"
			mock.AssertExec("podman", "rm", "-f", "silo-abc12345")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Container not found is a no-op", func(t *testing.T) {
		t.Run("Scenario: missing container prints not found and exits 0", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("false"),
				"podman container exists silo-abc12345":                              exec.Command("false"),
			})

			// When I run `silo rm`
			err := cmd.Remove(nil)

			// Then the output should contain "silo-abc12345 not found"
			output := internal.CaptureStdout(func() { cmd.Remove(nil) })
			if !strings.Contains(output, "silo-abc12345 not found") {
				t.Errorf("expected output to contain 'silo-abc12345 not found', got: %s", output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: rm does not remove the image", func(t *testing.T) {
		t.Run("Scenario: rm only removes the container, not the image", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman rm <any>":                                                   exec.Command("true"),
			})

			// When I run `silo rm`
			cmd.Remove(nil)

			// Then podman should run "rm" on "silo-abc12345"
			// But podman should not run "rmi" on "silo-abc12345"
			mock.AssertNoExec("podman", "rmi", "<any>")
		})
	})

	t.Run("Rule: Requires workspace to be initialized", func(t *testing.T) {
		t.Run("Scenario: rm fails when workspace is not initialized", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			dir := t.TempDir()
			orig, _ := os.Getwd()
			t.Cleanup(func() { os.Chdir(orig) })
			os.Chdir(dir)

			// When I run `silo rm`
			err := cmd.Remove(nil)

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