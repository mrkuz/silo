package features_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo exec — Run a command in the workspace container
// `silo exec` runs an arbitrary command inside the running workspace container.
// Unlike `silo connect`, it does not start the container or trigger the lifecycle
// chain — the container must already be running. It requires the workspace to have
// been initialized.
func TestFeatureExec(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Executes a command in the running container", func(t *testing.T) {
		t.Run("Scenario: exec runs the given command in the container", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
			})

			// When I run `silo exec -- echo hello`
			err := cmd.Exec([]string{"echo", "hello"})

			// Then podman should run "exec" with "-ti" on "silo-abc12345" with args "echo" "hello"
			execCall := mock.AssertExec("podman", "exec", "-ti", "silo-abc12345", "<...>")
			if execCall == nil {
				return
			}
			argsStr := strings.Join(execCall.Args, " ")
			if !strings.Contains(argsStr, "echo") || !strings.Contains(argsStr, "hello") {
				t.Errorf("expected exec to include 'echo' and 'hello', got: %s", argsStr)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Requires the container to be running", func(t *testing.T) {
		t.Run("Scenario: exec fails when container is not running", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
			})

			// When I run `silo exec echo hello`
			err := cmd.Exec([]string{"echo", "hello"})

			// Then the exit code should not be 0
			if err == nil {
				t.Errorf("expected error, got nil")
			}
			// And the error should indicate "silo-abc12345 is not running"
			if err != nil && !strings.Contains(err.Error(), "silo-abc12345 is not running") {
				t.Errorf("expected error to mention 'silo-abc12345 is not running', got: %v", err)
			}
		})

		t.Run("Scenario: exec does not start the container", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("false"),
			})

			// When I run `silo exec echo hello`
			err := cmd.Exec([]string{"echo", "hello"})

			// Then the exit code should not be 0
			if err == nil {
				t.Errorf("expected error, got nil")
			}
			// And no podman start should be called
			mock.AssertNoExec("podman", "start", "<any>")
		})
	})

	t.Run("Rule: Requires workspace to be initialized", func(t *testing.T) {
		t.Run("Scenario: exec fails when workspace is not initialized", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			dir := t.TempDir()
			orig, _ := os.Getwd()
			t.Cleanup(func() { os.Chdir(orig) })
			os.Chdir(dir)

			// When I run `silo exec echo hello`
			err := cmd.Exec([]string{"echo", "hello"})

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