package features_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo create — Create a workspace container
// `silo create` builds images if needed and creates a Podman container from the
// workspace image. The container is left stopped. Subsequent runs skip creation
// if the container already exists.
func TestFeatureCreate(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory
	// and the user image "silo-alice" exists
	// and the workspace image "silo-abc12345" exists

	t.Run("Rule: Creates the container from the workspace image", func(t *testing.T) {
		t.Run("Scenario: create makes the container", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo create`
			err := cmd.Create(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then a Podman container "silo-abc12345" should be created
			mock.AssertExec("podman", "create", "<...>")
			// And the exit code should be 0
		})

		t.Run("Scenario: create does not start the container", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo create`
			err := cmd.Create(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the container "silo-abc12345" should exist but not be running
			mock.AssertExec("podman", "container", "exists", "silo-abc12345")
			mock.AssertNoExec("podman", "start", "<any>")
		})

		t.Run("Scenario: create prints a creation message", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo create`
			output := internal.CaptureStdout(func() { cmd.Create(nil) })

			// Then the output should contain "Creating silo-abc12345..."
			if !strings.Contains(output, "Creating silo-abc12345...") {
				t.Errorf("expected output to contain 'Creating silo-abc12345...', got: %s", output)
			}
		})
	})

	t.Run("Rule: Idempotency — container is not recreated if it already exists", func(t *testing.T) {
		t.Run("Scenario: existing container is not overwritten", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("true"),
			})

			// When I run `silo create`
			output := internal.CaptureStdout(func() { cmd.Create(nil) })

			// Then the output should contain "silo-abc12345 already exists"
			if !strings.Contains(output, "silo-abc12345 already exists") {
				t.Errorf("expected output to contain 'silo-abc12345 already exists', got: %s", output)
			}
			// And no new container should be created
			mock.AssertNoExec("podman", "create", "<any>")
			// And the exit code should be 0
		})
	})

	t.Run("Rule: --dry-run prints the podman create command without running it", func(t *testing.T) {
		t.Run("Scenario: dry-run shows full podman create command", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo create --dry-run`
			output := internal.CaptureStdout(func() { cmd.Create([]string{"--dry-run"}) })

			// Then the output should contain "podman create"
			if !strings.Contains(output, "podman create") {
				t.Errorf("expected output to contain 'podman create', got: %s", output)
			}
			// And the output should contain "--name" and "silo-abc12345"
			if !strings.Contains(output, "--name") || !strings.Contains(output, "silo-abc12345") {
				t.Errorf("expected output to contain '--name' and 'silo-abc12345', got: %s", output)
			}
			// And the output should contain "--hostname" and "silo-abc12345"
			if !strings.Contains(output, "--hostname") || !strings.Contains(output, "silo-abc12345") {
				t.Errorf("expected output to contain '--hostname' and 'silo-abc12345', got: %s", output)
			}
		})

		t.Run("Scenario: dry-run does not create a container", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo create --dry-run`
			err := cmd.Create([]string{"--dry-run"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the container "silo-abc12345" should not exist
			mock.AssertNoExec("podman", "create", "<any>")
			// And the exit code should be 0
		})

		t.Run("Scenario: dry-run shows workspace mount", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo create --dry-run`
			output := internal.CaptureStdout(func() { cmd.Create([]string{"--dry-run"}) })

			// Then the output should contain "--volume"
			if !strings.Contains(output, "--volume") {
				t.Errorf("expected output to contain '--volume', got: %s", output)
			}
			// And the output should contain "--workdir"
			if !strings.Contains(output, "--workdir") {
				t.Errorf("expected output to contain '--workdir', got: %s", output)
			}
		})

		t.Run("Scenario: dry-run works even if container already exists", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("true"),
			})

			// When I run `silo create --dry-run`
			output := internal.CaptureStdout(func() { cmd.Create([]string{"--dry-run"}) })

			// Then the output should contain "podman create"
			if !strings.Contains(output, "podman create") {
				t.Errorf("expected output to contain 'podman create', got: %s", output)
			}
			// And the output should contain "--name" and "silo-abc12345"
			if !strings.Contains(output, "--name") || !strings.Contains(output, "silo-abc12345") {
				t.Errorf("expected output to contain '--name' and 'silo-abc12345', got: %s", output)
			}
			// And no new container should be created (ContainerExists not called in dry-run mode)
			mock.AssertNoExec("podman", "create", "<any>")
			// And the exit code should be 0
		})
	})

	t.Run("Rule: Builds images if missing before creating container", func(t *testing.T) {
		t.Run("Scenario: missing user image triggers build", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("false"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo create`
			err := cmd.Create(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the user image "silo-alice" should be built
			// Note: podman build uses RunVisible which outputs to stdout
			mock.AssertNoExec("podman", "create", "<any>")
		})

		t.Run("Scenario: missing workspace image triggers build", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("false"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo create`
			err := cmd.Create(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the workspace image "silo-abc12345" should be built
			mock.AssertExec("podman", "build", "<...>")
			// And a Podman container "silo-abc12345" should be created
			mock.AssertExec("podman", "create", "<...>")
		})
	})

	t.Run("Rule: Uses create arguments from config", func(t *testing.T) {
		t.Run("Scenario: podman-enabled config passes security-opt and device flags", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.Features.Podman = true
			cfg.Create.Arguments = []string{"--security-opt", "label=disable", "--device", "/dev/fuse"}
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo create`
			output := internal.CaptureStdout(func() { cmd.Create([]string{"--dry-run"}) })

			// Then the podman create command should include "--security-opt"
			if !strings.Contains(output, "--security-opt") {
				t.Errorf("expected output to contain '--security-opt', got: %s", output)
			}
			// And the podman create command should include "label=disable"
			if !strings.Contains(output, "label=disable") {
				t.Errorf("expected output to contain 'label=disable', got: %s", output)
			}
			// And the podman create command should include "--device"
			if !strings.Contains(output, "--device") {
				t.Errorf("expected output to contain '--device', got: %s", output)
			}
			// And the podman create command should include "/dev/fuse"
			if !strings.Contains(output, "/dev/fuse") {
				t.Errorf("expected output to contain '/dev/fuse', got: %s", output)
			}
		})

		t.Run("Scenario: podman-disabled config passes cap-drop, cap-add, and security-opt flags", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.Features.Podman = false
			cfg.Create.Arguments = []string{"--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges"}
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo create`
			output := internal.CaptureStdout(func() { cmd.Create([]string{"--dry-run"}) })

			// Then the podman create command should include "--cap-drop=ALL"
			if !strings.Contains(output, "--cap-drop=ALL") {
				t.Errorf("expected output to contain '--cap-drop=ALL', got: %s", output)
			}
			// And the podman create command should include "--cap-add=NET_BIND_SERVICE"
			if !strings.Contains(output, "--cap-add=NET_BIND_SERVICE") {
				t.Errorf("expected output to contain '--cap-add=NET_BIND_SERVICE', got: %s", output)
			}
			// And the podman create command should include "--security-opt" and "no-new-privileges"
			if !strings.Contains(output, "--security-opt") || !strings.Contains(output, "no-new-privileges") {
				t.Errorf("expected output to contain '--security-opt' and 'no-new-privileges', got: %s", output)
			}
		})

		t.Run("Scenario: custom create arguments from config are passed to podman", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.Create.Arguments = []string{"--memory=2g", "--cpus=4"}
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman create <...>":                   exec.Command("true"),
			})

			// When I run `silo create`
			err := cmd.Create(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the podman create command should include "--memory=2g"
			execCall := mock.AssertExec("podman", "create", "<...>")
			if execCall == nil {
				return
			}
			argsStr := strings.Join(execCall.Args, " ")
			if !strings.Contains(argsStr, "--memory=2g") {
				t.Errorf("expected podman create to include '--memory=2g', got: %s", argsStr)
			}
			// And the podman create command should include "--cpus=4"
			if !strings.Contains(argsStr, "--cpus=4") {
				t.Errorf("expected podman create to include '--cpus=4', got: %s", argsStr)
			}
		})
	})

	t.Run("Rule: Shared volume mounts when feature is enabled", func(t *testing.T) {
		t.Run("Scenario: shared volume is mounted when enabled", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.Features.SharedVolume = true
			cfg.SharedVolume.Paths = []string{"/silo/shared"}
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman create <...>":                   exec.Command("true"),
			})

			// When I run `silo create`
			err := cmd.Create(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the podman create command should include "--mount" with "type=volume" and "source=silo-shared" and "target=/silo/shared" and ",Z"
			execCall := mock.AssertExec("podman", "create", "<...>")
			if execCall == nil {
				return
			}
			argsStr := strings.Join(execCall.Args, " ")
			if !strings.Contains(argsStr, "--mount") {
				t.Errorf("expected podman create to include '--mount', got: %s", argsStr)
			}
			if !strings.Contains(argsStr, "type=volume") {
				t.Errorf("expected podman create to include 'type=volume', got: %s", argsStr)
			}
			if !strings.Contains(argsStr, "source=silo-shared") {
				t.Errorf("expected podman create to include 'source=silo-shared', got: %s", argsStr)
			}
			if !strings.Contains(argsStr, "target=/silo/shared") {
				t.Errorf("expected podman create to include 'target=/silo/shared', got: %s", argsStr)
			}
			if !strings.Contains(argsStr, ",Z") {
				t.Errorf("expected podman create to include ',Z', got: %s", argsStr)
			}
		})

		t.Run("Scenario: shared volume paths are mounted as subpath volumes", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.Features.SharedVolume = true
			cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman create <...>":                   exec.Command("true"),
			})

			// When I run `silo create`
			err := cmd.Create(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the podman create command should include "--mount" with "type=volume,source=silo-shared,target=/silo/shared,subpath=home/alice/.cache/uv,Z"
			execCall := mock.AssertExec("podman", "create", "<...>")
			if execCall == nil {
				return
			}
			argsStr := strings.Join(execCall.Args, " ")
			if !strings.Contains(argsStr, "subpath=home/alice/.cache/uv") {
				t.Errorf("expected podman create to include 'subpath=home/alice/.cache/uv', got: %s", argsStr)
			}
		})

		t.Run("Scenario: shared volume is not mounted when disabled", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.Features.SharedVolume = false
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman create <...>":                   exec.Command("true"),
			})

			// When I run `silo create`
			err := cmd.Create(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the podman create command should not include "/silo/shared"
			execCall := mock.AssertExec("podman", "create", "<...>")
			if execCall == nil {
				return
			}
			argsStr := strings.Join(execCall.Args, " ")
			if strings.Contains(argsStr, "/silo/shared") {
				t.Errorf("expected podman create to not include '/silo/shared', got: %s", argsStr)
			}
		})
	})
}
