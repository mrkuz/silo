package features_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo rm — Remove the workspace image
// `silo rm` removes the workspace image. If the container exists and is stopped,
// it is removed first. If the container is running, an error is returned and
// neither the container nor the image is touched. Unlike `silo user rm`, this
// removes the per-workspace image (`silo-<id>`), not the shared user image
// (`silo-<user>`).
func TestFeatureRm(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Removes the workspace image", func(t *testing.T) {
		t.Run("Scenario: rm removes the workspace image when no container exists", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman rmi silo-abc12345":              exec.Command("true"),
			})

			// When I run `silo rm`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.Remove()
			})

			// Then podman should run "rmi" on "silo-abc12345"
			mock.AssertExec("podman", "rmi", "silo-abc12345")
			// And the output should contain "Removing silo-abc12345..."
			if !strings.Contains(output, "Removing silo-abc12345...") {
				t.Errorf("expected 'Removing silo-abc12345...' in output, got: %s", output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: missing image prints not found", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman image exists silo-abc12345":     exec.Command("false"),
			})

			// When I run `silo rm`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.Remove()
			})

			// Then the output should contain "silo-abc12345 not found"
			if !strings.Contains(output, "silo-abc12345 not found") {
				t.Errorf("expected 'silo-abc12345 not found' in output, got: %s", output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Running container blocks removal", func(t *testing.T) {
		t.Run("Scenario: running container returns error without modifying state", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
			})

			// When I run `silo rm`
			err := cmd.Remove()

			// Then the error should contain "silo-abc12345 is running"
			if err == nil {
				t.Fatal("expected error when container is running")
			}
			if !strings.Contains(err.Error(), "silo-abc12345 is running") {
				t.Errorf("expected error to contain 'silo-abc12345 is running', got: %v", err)
			}
			// And podman should not run "stop" on "silo-abc12345"
			mock.AssertNoExec("podman", "stop", "<any>")
			// And podman should not run "rm" on "silo-abc12345"
			mock.AssertNoExec("podman", "rm", "<any>")
			// And podman should not run "rmi" on "silo-abc12345"
			mock.AssertNoExec("podman", "rmi", "<any>")
			// And the exit code should not be 0
			if err == nil {
				t.Error("expected non-zero exit code when container is running")
			}
		})
	})

	t.Run("Rule: Stopped container is removed before image removal", func(t *testing.T) {
		t.Run("Scenario: stopped container is removed before image removal", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
				"podman container ls --format json --filter name=silo-abc12345":      exec.Command("echo", "[]"),
				"podman rm -f silo-abc12345":                                         exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
				"podman rmi silo-abc12345":                                           exec.Command("true"),
			})

			// When I run `silo rm`
			err := cmd.Remove()

			// Then podman should run "rm" on "silo-abc12345"
			mock.AssertExec("podman", "rm", "-f", "silo-abc12345")
			// And podman should run "rmi" on "silo-abc12345"
			mock.AssertExec("podman", "rmi", "silo-abc12345")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Requires workspace to be initialized", func(t *testing.T) {
		t.Run("Scenario: rm fails when workspace is not initialized", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			base := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", base)
			dir := t.TempDir()
			orig, _ := os.Getwd()
			t.Cleanup(func() { os.Chdir(orig) })
			os.Chdir(dir)

			// When I run `silo rm`
			err := cmd.Remove()

			// Then the exit code should not be 0
			// And the error should indicate ".silo/silo.toml" is missing
			if err == nil {
				t.Error("expected error when workspace is not initialized")
			}
		})
	})

	t.Run("Rule: rm does not remove the user image", func(t *testing.T) {
		t.Run("Scenario: rm only removes the workspace image, not the user image", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman image exists silo-alice":        exec.Command("true"),
				"podman rmi silo-abc12345":              exec.Command("true"),
			})

			// When I run `silo rm`
			err := cmd.Remove()

			// Then podman should run "rmi" on "silo-abc12345"
			mock.AssertExec("podman", "rmi", "silo-abc12345")
			// But podman should not run "rmi" on "silo-alice"
			mock.AssertNoExec("podman", "rmi", "silo-alice")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})
}
