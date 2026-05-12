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
			internal.SubsequentRun(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
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
			internal.SubsequentRun(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
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
			internal.SubsequentRun(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
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
			internal.SubsequentRun(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
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

		t.Run("Scenario: missing image triggers build before container creation", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			internal.SubsequentRun(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman image exists silo-abc12345":     exec.Command("false"),
				"podman build -t silo-abc12345 <...>":   exec.Command("true"),
				"podman create <...>":                   exec.Command("true"),
				"podman start silo-abc12345":            exec.Command("true"),
			})

			// When I run `silo start`
			err := cmd.Start()

			// Then workspace image should be built
			mock.AssertExec("podman", "build", "-t", "silo-abc12345", "<...>")
			mock.AssertExec("podman", "create", "<...>")
			mock.AssertExec("podman", "start", "silo-abc12345")
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: workspace ports are passed to podman create", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.Network.Ports = []string{"8080:8080"}
			internal.SubsequentRun(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman create <...>":                   exec.Command("true"),
				"podman start silo-abc12345":            exec.Command("true"),
			})

			// When I run `silo start`
			err := cmd.Start()

			// Then podman create should include "-p 8080:8080"
			rec := mock.AssertExec("podman", "create", "<...>")
			if rec != nil {
				if !strings.Contains(rec.String(), "-p 8080:8080") {
					t.Errorf("expected -p 8080:8080 in create command, got: %s", rec.String())
				}
			}
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: multiple ports are all passed to podman create", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.Network.Ports = []string{"8080:8080", "3000:3000"}
			internal.SubsequentRun(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman create <...>":                   exec.Command("true"),
				"podman start silo-abc12345":            exec.Command("true"),
			})

			// When I run `silo start`
			err := cmd.Start()

			// Then podman create should include both port mappings
			rec := mock.AssertExec("podman", "create", "<...>")
			if rec != nil {
				recStr := rec.String()
				if !strings.Contains(recStr, "-p 8080:8080") || !strings.Contains(recStr, "-p 3000:3000") {
					t.Errorf("expected -p 8080:8080 and -p 3000:3000 in create command, got: %s", recStr)
				}
			}
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: empty ports array adds no -p arguments", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.Network.Ports = []string{}
			internal.SubsequentRun(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman create <...>":                   exec.Command("true"),
				"podman start silo-abc12345":            exec.Command("true"),
			})

			// When I run `silo start`
			err := cmd.Start()

			// Then podman create should not include any -p arguments
			rec := mock.AssertExec("podman", "create", "<...>")
			if rec != nil {
				if strings.Contains(rec.String(), "-p") {
					t.Errorf("expected no -p arguments in create command, got: %s", rec.String())
				}
			}
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Runs volume setup before starting", func(t *testing.T) {
		t.Run("Scenario: shared volume directories are created before container starts", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.Persistence.SharedPaths = []string{"$HOME/.cache/uv/"}
			internal.SubsequentRun(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
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
			expectedPath := "/silo/persistence/shared/home/alice/.cache/uv"
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
			internal.SubsequentRun(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
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
