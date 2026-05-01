package features_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo devcontainer rm — Remove the devcontainer
// `silo devcontainer rm` removes the devcontainer. It refuses to remove a running
// devcontainer unless `--force` is given. It does not affect the workspace container.
func TestFeatureDevcontainerRm(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Removes a stopped devcontainer", func(t *testing.T) {
		t.Run("Scenario: rm removes stopped devcontainer", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "false"),
				"podman container exists silo-abc12345-dev":                              exec.Command("true"),
				"podman rm -f silo-abc12345-dev":                                         exec.Command("true"),
			})

			// When I run `silo devcontainer rm`
			output := internal.CaptureStdout(func() { cmd.DevcontainerRemove(nil) })

			// Then podman should run "rm" with "-f" on "silo-abc12345-dev"
			mock.AssertExec("podman", "rm", "-f", "silo-abc12345-dev")
			// And the output should contain "Removing silo-abc12345-dev..."
			if !strings.Contains(output, "Removing silo-abc12345-dev...") {
				t.Errorf("expected output to contain 'Removing silo-abc12345-dev...', got: %s", output)
			}
		})
	})

	t.Run("Rule: Refuses to remove a running devcontainer without --force", func(t *testing.T) {
		t.Run("Scenario: rm without --force returns an error", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
				"podman container exists silo-abc12345-dev":                              exec.Command("true"),
			})

			// When I run `silo devcontainer rm`
			err := cmd.DevcontainerRemove(nil)

			// Then the exit code should not be 0
			if err == nil {
				t.Errorf("expected error, got nil")
			}
			// And the error should indicate "silo-abc12345-dev is running"
			if err != nil && !strings.Contains(err.Error(), "silo-abc12345-dev is running") {
				t.Errorf("expected error to mention 'silo-abc12345-dev is running', got: %v", err)
			}
			// And the devcontainer "silo-abc12345-dev" should still exist
			mock.AssertExec("podman", "container", "exists", "silo-abc12345-dev")
		})
	})

	t.Run("Rule: --force stops and removes a running devcontainer", func(t *testing.T) {
		t.Run("Scenario: --force stops running devcontainer before removing", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
				"podman container exists silo-abc12345-dev":                              exec.Command("true"),
				"podman stop -t 0 silo-abc12345-dev":                                     exec.Command("true"),
				"podman rm -f silo-abc12345-dev":                                         exec.Command("true"),
			})

			// When I run `silo devcontainer rm --force`
			err := cmd.DevcontainerRemove([]string{"--force"})

			// Then podman should run "stop" with "-t" and "0" on "silo-abc12345-dev"
			mock.AssertExec("podman", "stop", "-t", "0", "silo-abc12345-dev")
			// And podman should run "rm" with "-f" on "silo-abc12345-dev"
			mock.AssertExec("podman", "rm", "-f", "silo-abc12345-dev")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Devcontainer not found is a no-op", func(t *testing.T) {
		t.Run("Scenario: missing devcontainer prints not found and exits 0", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("false"),
				"podman container exists silo-abc12345-dev":                              exec.Command("false"),
			})

			// When I run `silo devcontainer rm`
			err := cmd.DevcontainerRemove(nil)

			// Then the output should contain "silo-abc12345-dev not found"
			output := internal.CaptureStdout(func() { cmd.DevcontainerRemove(nil) })
			if !strings.Contains(output, "silo-abc12345-dev not found") {
				t.Errorf("expected output to contain 'silo-abc12345-dev not found', got: %s", output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Requires workspace to be initialized", func(t *testing.T) {
		t.Run("Scenario: devcontainer rm fails when workspace is not initialized", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			dir := t.TempDir()
			orig, _ := os.Getwd()
			t.Cleanup(func() { os.Chdir(orig) })
			os.Chdir(dir)

			// When I run `silo devcontainer rm`
			err := cmd.DevcontainerRemove(nil)

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

	t.Run("Rule: Does not stop or remove the workspace container or image", func(t *testing.T) {
		t.Run("Scenario: rm only removes the devcontainer, not the workspace container", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "false"),
				"podman container inspect --format {{.State.Running}} silo-abc12345":     exec.Command("echo", "true"),
				"podman container exists silo-abc12345-dev":                              exec.Command("true"),
				"podman rm <any>": exec.Command("true"),
			})

			// When I run `silo devcontainer rm`
			cmd.DevcontainerRemove(nil)

			// Then podman should run "rm" on "silo-abc12345-dev"
			// But podman should not run "stop" on "silo-abc12345"
			mock.AssertNoExec("podman", "stop", "silo-abc12345")
			// And podman should not run "rm" on "silo-abc12345"
			mock.AssertNoExec("podman", "rm", "silo-abc12345")
		})
		t.Run("Scenario: rm only removes the devcontainer, not the workspace image", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "false"),
				"podman container exists silo-abc12345-dev":                              exec.Command("true"),
				"podman rm <any>": exec.Command("true"),
			})

			// When I run `silo devcontainer rm`
			cmd.DevcontainerRemove(nil)

			// Then podman should run "rm" on "silo-abc12345-dev"
			// But podman should not run "rmi" on "silo-abc12345"
			mock.AssertNoExec("podman", "rmi", "silo-abc12345")
		})
	})
}
