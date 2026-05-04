package features_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo start — Start the workspace container
// `silo start` ensures the container is running. It builds images and creates
// the container if needed, then starts it. If the container is already running,
// it is a no-op. Unlike the default silo invocation, it does not attach to the
// container — it returns after starting.
func TestFeatureStart(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Starts the container", func(t *testing.T) {
		t.Run("Scenario: start runs podman start", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
				"podman start silo-abc12345":                                         exec.Command("true"),
			})

			// When I run `silo start`
			err := cmd.Start()

			// Then podman should run "start" on "silo-abc12345"
			mock.AssertExec("podman", "start", "silo-abc12345")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: start prints a message when starting", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
				"podman start silo-abc12345":                                         exec.Command("true"),
			})

			// When I run `silo start`
			output := internal.CaptureStdout(func() {
				cmd.Start()
			})

			// Then the output should contain "Starting silo-abc12345..."
			if !strings.Contains(output, "Starting silo-abc12345...") {
				t.Errorf("expected 'Starting silo-abc12345...' in output, got: %s", output)
			}
		})
	})

	t.Run("Rule: Idempotency — already running container is a no-op", func(t *testing.T) {
		t.Run("Scenario: running container is not restarted", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
			})

			// When I run `silo start`
			err := cmd.Start()

			// Then no podman start should be called
			mock.AssertNoExec("podman", "start", "<any>")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Creates container if missing (builds images if needed)", func(t *testing.T) {
		t.Run("Scenario: missing container triggers full build-and-create chain", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman create <...>":                   exec.Command("true"),
				"podman start silo-abc12345":            exec.Command("true"),
			})

			// When I run `silo start`
			err := cmd.Start()

			// Then the container "silo-abc12345" should be created
			mock.AssertExec("podman", "create", "<...>")
			// And the container "silo-abc12345" should be running
			mock.AssertExec("podman", "start", "silo-abc12345")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: missing images trigger build before container creation", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman image exists silo-alice":        exec.Command("false"),
				"podman image exists silo-abc12345":     exec.Command("false"),
				"podman build -t silo-alice <...>":      exec.Command("true"),
				"podman build -t silo-abc12345 <...>":   exec.Command("true"),
				"podman create <...>":                   exec.Command("true"),
				"podman start silo-abc12345":            exec.Command("true"),
			})

			// When I run `silo start`
			err := cmd.Start()

			// Then both user and workspace images should be built
			mock.AssertExec("podman", "build", "-t", "silo-alice", "<...>")
			mock.AssertExec("podman", "build", "-t", "silo-abc12345", "<...>")
			mock.AssertExec("podman", "create", "<...>")
			mock.AssertExec("podman", "start", "silo-abc12345")
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Runs volume setup before starting", func(t *testing.T) {
		t.Run("Scenario: shared volume directories are created before container starts", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
				"podman run --rm <...>":                                              exec.Command("true"),
				"podman start silo-abc12345":                                         exec.Command("true"),
			})

			// When I run `silo start`
			err := cmd.Start()

			// Then shared volume directories should be created before the container starts
			// Verify volume setup (podman run --rm) was called
			record := mock.AssertExec("podman", "run", "--rm", "<...>")
			cmdStr := strings.Join(record.Args, " ")
			expectedPath := "/silo/shared/home/alice/.cache/uv"
			if !strings.Contains(cmdStr, "mkdir -p "+expectedPath) {
				t.Errorf("expected mkdir -p %s, got: %s", expectedPath, cmdStr)
			}
			// Verify container was started
			mock.AssertExec("podman", "start", "silo-abc12345")
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Does not connect to the container", func(t *testing.T) {
		t.Run("Scenario: start does not attach to the container", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
				"podman start silo-abc12345":                                         exec.Command("true"),
			})

			// When I run `silo start`
			err := cmd.Start()

			// Then no podman exec should be called
			mock.AssertNoExec("podman", "exec", "<any>")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})
}
