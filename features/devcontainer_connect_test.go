package features_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo devcontainer connect — Open an interactive shell in the devcontainer
// `silo devcontainer connect` opens an interactive shell session inside the
// running devcontainer. The devcontainer is named `<workspace-container-name>-dev`.
// It requires the devcontainer to exist and be running. It does not accept any arguments.
func TestFeatureDevcontainerConnect(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Connects to the running devcontainer", func(t *testing.T) {
		t.Run("Scenario: devcontainer connect opens an interactive shell", func(t *testing.T) {
			// Given the devcontainer "silo-abc12345-dev" is running
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345-dev":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
			})

			// When I run `silo devcontainer connect`
			err := cmd.DevcontainerConnect()

			// Then podman should run "exec" with "-ti" on "silo-abc12345-dev"
			// And the command should be "/bin/sh"
			// And the exit code should be 0
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			execRecord := mock.AssertExec("podman", "exec", "-ti", "silo-abc12345-dev", "/bin/sh")
			if execRecord == nil {
				t.Error("expected exec to be called")
			}
		})

		t.Run("Scenario: devcontainer connect prints a message before opening shell", func(t *testing.T) {
			// Given the devcontainer "silo-abc12345-dev" is running
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345-dev":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
			})

			// When I run `silo devcontainer connect`
			output := internal.CaptureStdout(func() { cmd.DevcontainerConnect() })

			// Then the output should contain "Connecting to silo-abc12345-dev..."
			if !strings.Contains(output, "Connecting to silo-abc12345-dev...") {
				t.Errorf("expected output to contain 'Connecting to silo-abc12345-dev...', got: %s", output)
			}
		})
	})

	t.Run("Rule: Requires devcontainer to be running", func(t *testing.T) {
		t.Run("Scenario: devcontainer connect fails if devcontainer is not running", func(t *testing.T) {
			// Given the devcontainer "silo-abc12345-dev" exists but is stopped
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345-dev":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "false"),
			})

			// When I run `silo devcontainer connect`
			err := cmd.DevcontainerConnect()

			// Then the exit code should not be 0
			// And the error should contain "not running"
			if err == nil {
				t.Fatal("expected error when devcontainer is not running")
			}
			if !strings.Contains(err.Error(), "not running") {
				t.Errorf("expected error about devcontainer not running, got: %v", err)
			}
		})

		t.Run("Scenario: devcontainer connect fails if devcontainer does not exist", func(t *testing.T) {
			// Given no devcontainer exists
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345-dev": exec.Command("false"),
			})

			// When I run `silo devcontainer connect`
			err := cmd.DevcontainerConnect()

			// Then the exit code should not be 0
			// And the error should contain "not found" or "does not exist"
			if err == nil {
				t.Fatal("expected error when devcontainer does not exist")
			}
			if !strings.Contains(err.Error(), "does not exist") {
				t.Errorf("expected error about devcontainer not existing, got: %v", err)
			}
		})
	})

	t.Run("Rule: Exiting the shell leaves the devcontainer running", func(t *testing.T) {
		t.Run("Scenario: exiting the devcontainer connect shell does not stop the devcontainer", func(t *testing.T) {
			// Given the devcontainer "silo-abc12345-dev" is running
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345-dev":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
			})

			// When I run `silo devcontainer connect`
			// And the interactive session ends
			err := cmd.DevcontainerConnect()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the devcontainer "silo-abc12345-dev" should still be running
			mock.AssertExec("podman", "container", "inspect", "--format", "{{.State.Running}}", "silo-abc12345-dev")
			// And podman should not run "stop" on "silo-abc12345-dev"
			mock.AssertNoExec("podman", "stop", "<any>")
		})
	})

	t.Run("Rule: Multiple sessions can be connected simultaneously", func(t *testing.T) {
		t.Run("Scenario: two parallel devcontainer connect calls create two independent shells", func(t *testing.T) {
			// Given the devcontainer "silo-abc12345-dev" is running
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345-dev":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
			})

			// When I run `silo devcontainer connect` and `silo devcontainer connect` in parallel
			err := cmd.DevcontainerConnect()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then two independent shell sessions should be opened in "silo-abc12345-dev"
			execCalls := mock.AssertExec("podman", "exec", "-ti", "silo-abc12345-dev", "<...>")
			if execCalls == nil {
				t.Error("expected at least one exec call")
			}

			mock.Reset()
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345-dev":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
			})

			err = cmd.DevcontainerConnect()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Verify second connect also made an exec call
			secondCall := mock.AssertExec("podman", "exec", "-ti", "silo-abc12345-dev", "<...>")
			if secondCall == nil {
				t.Error("expected second exec call for parallel connect")
			}
		})
	})

	t.Run("Rule: Does not affect the workspace container", func(t *testing.T) {
		t.Run("Scenario: devcontainer connect does not check workspace container state", func(t *testing.T) {
			// Given the devcontainer "silo-abc12345-dev" is running
			// And no workspace container exists
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345-dev":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
			})

			// When I run `silo devcontainer connect`
			err := cmd.DevcontainerConnect()

			// Then podman should run "exec" with "-ti" on "silo-abc12345-dev"
			// And the workspace container state should not cause failure
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			mock.AssertExec("podman", "exec", "-ti", "silo-abc12345-dev", "/bin/sh")
			// Workspace container (silo-abc12345) should not be checked
			mock.AssertNoExec("podman", "container", "exists", "silo-abc12345")
		})
	})
}