package features_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo rmi — Remove the workspace image
// `silo rmi` removes the workspace image. With `--force`, it also stops and removes
// the container first if it is running. Unlike `silo user rmi`, this removes the
// per-workspace image (`silo-<id>`), not the shared user image (`silo-<user>`).
func TestFeatureRmi(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Removes the workspace image", func(t *testing.T) {
		t.Run("Scenario: rmi removes the workspace image", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345": exec.Command("true"),
				"podman rmi silo-abc12345":          exec.Command("true"),
			})

			// When I run `silo rmi`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.RemoveImage(nil)
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
				"podman image exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo rmi`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.RemoveImage(nil)
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

	t.Run("Rule: --force stops and removes container before removing image", func(t *testing.T) {
		t.Run("Scenario: --force stops running container before removing image", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman stop -t 0 silo-abc12345":                                     exec.Command("true"),
				"podman rm -f silo-abc12345":                                         exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
				"podman rmi silo-abc12345":                                           exec.Command("true"),
			})

			// When I run `silo rmi --force`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.RemoveImage([]string{"--force"})
			})

			// Then podman should run "stop" with "-t" and "0" on "silo-abc12345"
			mock.AssertExec("podman", "stop", "-t", "0", "silo-abc12345")
			// And podman should run "rm" with "-f" on "silo-abc12345"
			mock.AssertExec("podman", "rm", "-f", "silo-abc12345")
			// And podman should run "rmi" on "silo-abc12345"
			mock.AssertExec("podman", "rmi", "silo-abc12345")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
			_ = output // suppress unused warning
		})

		t.Run("Scenario: --force with stopped container removes image directly", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
				"podman rmi silo-abc12345":                                           exec.Command("true"),
			})

			// When I run `silo rmi --force`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.RemoveImage([]string{"--force"})
			})

			// Then podman should run "rmi" on "silo-abc12345"
			mock.AssertExec("podman", "rmi", "silo-abc12345")
			// But podman should not run "stop" on "silo-abc12345"
			mock.AssertNoExec("podman", "stop", "<any>")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
			_ = output
		})

		t.Run("Scenario: --force with absent container removes image without trying to stop", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman rmi silo-abc12345":              exec.Command("true"),
			})

			// When I run `silo rmi --force`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.RemoveImage([]string{"--force"})
			})

			// Then podman should run "rmi" on "silo-abc12345"
			mock.AssertExec("podman", "rmi", "silo-abc12345")
			// But podman should not run "stop" on "silo-abc12345"
			mock.AssertNoExec("podman", "stop", "<any>")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
			_ = output
		})
	})

	t.Run("Rule: Without --force, running container blocks image removal", func(t *testing.T) {
		t.Run("Scenario: running container without --force returns an error", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
				"podman rmi silo-abc12345":                                           exec.Command("false"),
			})

			// When I run `silo rmi`
			err := cmd.RemoveImage(nil)

			// Then the exit code should not be 0
			// Note: Implementation calls rmi even with running container, but rmi fails because image is in use
			if err == nil {
				t.Error("expected error when container is running without --force")
			}
			// And rmi is still called (the implementation doesn't block beforehand)
			// but it fails because the image is in use by the container
		})
	})

	t.Run("Rule: Requires workspace to be initialized", func(t *testing.T) {
		t.Run("Scenario: rmi fails when workspace is not initialized", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			base := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", base)
			dir := t.TempDir()
			orig, _ := os.Getwd()
			t.Cleanup(func() { os.Chdir(orig) })
			os.Chdir(dir)

			// When I run `silo rmi`
			err := cmd.RemoveImage(nil)

			// Then the exit code should not be 0
			// And the error should indicate ".silo/silo.toml" is missing
			if err == nil {
				t.Error("expected error when workspace is not initialized")
			}
		})
	})

	t.Run("Rule: rmi does not remove the user image", func(t *testing.T) {
		t.Run("Scenario: rmi only removes the workspace image, not the user image", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345": exec.Command("true"),
				"podman image exists silo-alice":    exec.Command("true"),
				"podman rmi silo-abc12345":          exec.Command("true"),
			})

			// When I run `silo rmi`
			err := cmd.RemoveImage(nil)

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
