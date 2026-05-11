package features_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo build — Build workspace images
// `silo build` ensures the workspace image exists,
// building it if missing. It runs `silo init` implicitly first.
func TestFeatureBuild(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory
	// and the user's silo config directory has all starter files

	t.Run("Rule: Builds workspace image when missing", func(t *testing.T) {
		t.Run("Scenario: build creates workspace image", func(t *testing.T) {
			cfg := internal.MinimalConfig("abc12345")
			internal.SubsequentRun(t, cfg, "alice")

			// And no workspace image exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345":     exec.Command("false"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo build`
			if err := cmd.Build([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the workspace image "silo-abc12345" should be built
			workspaceBuild := mock.AssertExec("podman", "build", "-t", "silo-abc12345", "<...>")
			if workspaceBuild == nil {
				t.Fatal("expected workspace image to be built")
			}
		})
	})

	t.Run("Rule: Build outputs single image message", func(t *testing.T) {
		t.Run("Scenario: build prints build message", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			// And the user's XDG_CONFIG_HOME points to a fresh directory
			// And the user's silo config directory has all starter files
			cfg := internal.MinimalConfig("abc12345")
			internal.SubsequentRun(t, cfg, "alice")

			// And no workspace image exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345":     exec.Command("false"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo build`
			output := internal.CaptureStdout(func() { cmd.Build([]string{}) })

			// Then the output should contain "Building workspace image silo-abc12345..."
			workspaceIdx := strings.Index(output, "Building workspace image silo-abc12345...")
			if workspaceIdx < 0 {
				t.Errorf("expected output to contain 'Building workspace image silo-abc12345...', got: %s", output)
			}
		})
	})

	t.Run("Rule: Idempotency — existing images are skipped", func(t *testing.T) {
		t.Run("Scenario: workspace image exists is a no-op", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			// And the user's XDG_CONFIG_HOME points to a fresh directory
			// And the user's silo config directory has all starter files
			cfg := internal.MinimalConfig("abc12345")
			internal.SubsequentRun(t, cfg, "alice")

			// And the workspace image "silo-abc12345" exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo build`
			output := internal.CaptureStdout(func() { cmd.Build([]string{}) })

			// Then the output should contain "silo-abc12345 already exists"
			if !strings.Contains(output, "silo-abc12345 already exists") {
				t.Errorf("expected output to contain 'silo-abc12345 already exists', got: %s", output)
			}
			// And no build should occur
			mock.AssertNoExec("podman", "build", "<...>")
			// And the exit code should be 0 (implicit)
		})
	})

	t.Run("Rule: Init on demand — build initializes workspace if not initialized", func(t *testing.T) {
		t.Run("Scenario: build creates workspace config if missing", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			// And the user's XDG_CONFIG_HOME points to a fresh directory
			// And the user's silo config directory has all starter files
			internal.FirstRunWithFiles(t, map[string]string{
				"home.user.nix": internal.HomeUserNix,
				"silo.user.toml": `[general]
user = "alice"
`,
			})

			// Control the generated ID so we can verify exact names
			internal.SetGeneratedIDFunc(t, func() string { return "abc12345" })

			// And no workspace image exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists <any>":             exec.Command("false"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo build`
			if err := cmd.Build([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then a file ".silo/silo.toml" should be created
			if _, err := os.Stat(internal.SiloToml()); os.IsNotExist(err) {
				t.Error("expected .silo/silo.toml to be created")
			}
			// And the workspace image "silo-abc12345" should be built
			workspaceBuild := mock.AssertExec("podman", "build", "-t", "silo-abc12345", "<...>")
			if workspaceBuild == nil {
				t.Fatal("expected workspace image to be built")
			}
		})
	})

	t.Run("Rule: home.nix is baked into the workspace image", func(t *testing.T) {
		t.Run("Scenario: workspace home.nix content is included in the built image", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			cfg := internal.MinimalConfig("abc12345")
			internal.SubsequentRun(t, cfg, "alice")

			// And the workspace has "home.nix" with content:
			homeNix := filepath.Join(internal.SiloDir(), "home.nix")
			expectedContent := "home.packages = with pkgs; [ nodejs python3 ];\n"
			if err := os.WriteFile(homeNix, []byte(expectedContent), 0644); err != nil {
				t.Fatalf("write home.nix: %v", err)
			}

			// And no workspace image exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345":     exec.Command("false"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})
			mock.MockRead(map[string][]byte{
				homeNix: []byte(expectedContent),
			})

			// When I run `silo build`
			if err := cmd.Build([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the workspace image should be built
			mock.AssertExec("podman", "build", "-t", "silo-abc12345", "<...>")
			// And the workspace image build should include a file "home.nix" containing "nodejs python3"
			mock.AssertRead(homeNix)
		})
	})

	t.Run("Rule: --force forces workspace image rebuild", func(t *testing.T) {
		t.Run("Scenario: build --force rebuilds even when image exists", func(t *testing.T) {
			// Given the workspace image "silo-abc12345" exists
			// And the container "silo-abc12345" does not exist
			cfg := internal.MinimalConfig("abc12345")
			internal.SubsequentRun(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345":     exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo build --force`
			if err := cmd.Build([]string{"--force"}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the workspace image should be built with --no-cache
			mock.AssertExec("podman", "build", "-t", "silo-abc12345", "--no-cache", "<...>")
		})

		t.Run("Scenario: build --force aborts if container is running", func(t *testing.T) {
			// Given the workspace image "silo-abc12345" exists
			// And the container "silo-abc12345" is running
			cfg := internal.MinimalConfig("abc12345")
			internal.SubsequentRun(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345":                              exec.Command("true"),
				"podman container exists silo-abc12345":                         exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
			})

			// When I run `silo build --force`
			err := cmd.Build([]string{"--force"})

			// Then the exit code should not be 0
			// And the error should contain "running"
			if err == nil {
				t.Fatal("expected error when container is running")
			}
			if !strings.Contains(err.Error(), "running") {
				t.Errorf("expected error about running, got: %v", err)
			}
		})

		t.Run("Scenario: build --force aborts if container exists (stopped)", func(t *testing.T) {
			// Given the workspace image "silo-abc12345" exists
			// And the container "silo-abc12345" exists but is stopped
			cfg := internal.MinimalConfig("abc12345")
			internal.SubsequentRun(t, cfg, "alice")
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-abc12345":                                  exec.Command("true"),
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
			})

			// When I run `silo build --force`
			err := cmd.Build([]string{"--force"})

			// Then the exit code should not be 0
			// And the error should contain "exists"
			if err == nil {
				t.Fatal("expected error when container exists")
			}
			if !strings.Contains(err.Error(), "exists") {
				t.Errorf("expected error about exists, got: %v", err)
			}
		})
	})

	t.Run("Rule: unknown flag shows error and help", func(t *testing.T) {
		t.Run("Scenario: unknown flag shows error and help", func(t *testing.T) {
			internal.FirstRun(t)

			// When I run `silo build --unknown`
			err := cmd.Build([]string{"--unknown"})

			// Then the exit code should not be 0
			if err == nil {
				t.Fatal("expected error for unknown flag")
			}
			// And the error should contain "erroneous command"
			if !strings.Contains(err.Error(), `erroneous command`) {
				t.Errorf("expected erroneous command error, got: %v", err)
			}
			// And the error should contain the help text
			if !strings.Contains(err.Error(), "Usage:") {
				t.Errorf("expected help text in error, got: %v", err)
			}
		})
	})
}
