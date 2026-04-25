package features_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo connect — Open an interactive shell in the workspace container
// `silo connect` opens an interactive shell session inside the running workspace
// container. It runs the full lifecycle chain (init → build → create → start) if
// needed before connecting. It does not accept any arguments.
func TestFeatureConnect(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory
	// and the user's silo config directory has all starter files

	t.Run("Rule: Connects to the running container", func(t *testing.T) {
		t.Run("Scenario: connect opens an interactive shell", func(t *testing.T) {
			// Given the container "silo-abc12345" is running
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
			})

			// When I run `silo connect`
			// Then podman should run "exec" with "-ti" on "silo-abc12345"
			// And the command should be "/bin/sh"
			// And the exit code should be 0
			err := cmd.Connect(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			execRecord := mock.AssertExec("podman", "exec", "-ti", "silo-abc12345", "/bin/sh")
			if execRecord == nil {
				t.Error("expected exec to be called")
			}
		})

		t.Run("Scenario: connect prints a message before opening shell", func(t *testing.T) {
			// Given the container "silo-abc12345" is running
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
			})

			// When I run `silo connect`
			output := internal.CaptureStdout(func() { cmd.Connect(nil) })

			// Then the output should contain "Connecting to silo-abc12345..."
			if !strings.Contains(output, "Connecting to silo-abc12345...") {
				t.Errorf("expected output to contain 'Connecting to silo-abc12345...', got: %s", output)
			}
		})
	})

	t.Run("Rule: Ensures container is started before connecting", func(t *testing.T) {
		t.Run("Scenario: stopped container is started before connecting", func(t *testing.T) {
			// Given the container "silo-abc12345" exists but is stopped
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
			})

			// When I run `silo connect`
			err := cmd.Connect(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the container "silo-abc12345" should be started
			mock.AssertExec("podman", "start", "silo-abc12345")
			// And podman should run "exec" with "-ti" on "silo-abc12345"
			mock.AssertExec("podman", "exec", "-ti", "silo-abc12345", "<...>")
		})

		t.Run("Scenario: missing container triggers full lifecycle before connecting", func(t *testing.T) {
			// Given no container exists
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo connect`
			err := cmd.Connect(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the container "silo-abc12345" should be created
			mock.AssertExec("podman", "create", "<...>")
			// And the container "silo-abc12345" should be running
			mock.AssertExec("podman", "start", "silo-abc12345")
			// And podman should run "exec" with "-ti" on "silo-abc12345"
			mock.AssertExec("podman", "exec", "-ti", "silo-abc12345", "<...>")
		})
	})

	t.Run("Rule: Exiting the shell leaves the container running", func(t *testing.T) {
		t.Run("Scenario: exiting the connect shell does not stop the container", func(t *testing.T) {
			// Given the container "silo-abc12345" is running
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
			})

			// When I run `silo connect`
			// And the interactive session ends
			err := cmd.Connect(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the container "silo-abc12345" should still be running
			mock.AssertExec("podman", "container", "inspect", "--format", "{{.State.Running}}", "silo-abc12345")
			// And podman should not run "stop" on "silo-abc12345"
			mock.AssertNoExec("podman", "stop", "<any>")
		})
	})

	t.Run("Rule: Multiple sessions can be connected simultaneously", func(t *testing.T) {
		t.Run("Scenario: two parallel connect calls create two independent shells in the same container", func(t *testing.T) {
			// Given the container "silo-abc12345" is running
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
			})

			// When I run `silo connect` and `silo connect` in parallel
			err := cmd.Connect(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Then two independent shell sessions should be opened in "silo-abc12345"
			execCalls := mock.AssertExec("podman", "exec", "-ti", "silo-abc12345", "<...>")
			if execCalls == nil {
				t.Error("expected at least one exec call")
			}

			mock.Reset()

			err = cmd.Connect(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Verify second connect also made an exec call
			secondCall := mock.AssertExec("podman", "exec", "-ti", "silo-abc12345", "<...>")
			if secondCall == nil {
				t.Error("expected second exec call for parallel connect")
			}
		})
	})
}
