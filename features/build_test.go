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
// `silo build` ensures both the user image and the workspace image exist,
// building either or both if missing. It runs `silo init` implicitly first.
func TestFeatureBuild(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory
	// and the user's silo config directory has all starter files

	t.Run("Rule: Builds both images when both are missing", func(t *testing.T) {
		t.Run("Scenario: build creates user image first, then workspace image", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			// And the user's XDG_CONFIG_HOME points to a fresh directory
			// And the user's silo config directory has all starter files
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And no user image exists
			// And no workspace image exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":    exec.Command("false"),
				"podman image exists silo-abc12345": exec.Command("false"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo build`
			if err := cmd.Build([]string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the user image "silo-alice" should be built
			userBuild := mock.AssertExec("podman", "build", "-t", "silo-alice", "<...>")
			// And the workspace image "silo-abc12345" should be built
			workspaceBuild := mock.AssertExec("podman", "build", "-t", "silo-abc12345", "<...>")
			// And the user image should be built before the workspace image
			if userBuild.Seq >= workspaceBuild.Seq {
				t.Error("expected user image to be built before workspace image")
			}
			// And the exit code should be 0 (implicit)
		})

		t.Run("Scenario: build prints build messages in order", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			// And the user's XDG_CONFIG_HOME points to a fresh directory
			// And the user's silo config directory has all starter files
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And no user image exists
			// And no workspace image exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":    exec.Command("false"),
				"podman image exists silo-abc12345": exec.Command("false"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo build`
			output := internal.CaptureStdout(func() { cmd.Build([]string{}) })

			// Then the output should contain "Building user image silo-alice..."
			userIdx := strings.Index(output, "Building user image silo-alice...")
			if userIdx < 0 {
				t.Errorf("expected output to contain 'Building user image silo-alice...', got: %s", output)
			}
			// And the output should contain "Building workspace image silo-abc12345..."
			workspaceIdx := strings.Index(output, "Building workspace image silo-abc12345...")
			if workspaceIdx < 0 {
				t.Errorf("expected output to contain 'Building workspace image silo-abc12345...', got: %s", output)
			}
			if userIdx >= workspaceIdx {
				t.Error("expected 'Building user image' to appear before 'Building workspace image'")
			}
		})
	})

	t.Run("Rule: Idempotency — existing images are skipped", func(t *testing.T) {
		t.Run("Scenario: both images exist is a no-op", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			// And the user's XDG_CONFIG_HOME points to a fresh directory
			// And the user's silo config directory has all starter files
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":    exec.Command("true"),
				"podman image exists silo-abc12345": exec.Command("true"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo build`
			output := internal.CaptureStdout(func() { cmd.Build([]string{}) })

			// Then the output should contain "silo-abc12345 already exists"
			// Note: EnsureUserImage does not print when user image exists
			if !strings.Contains(output, "silo-abc12345 already exists") {
				t.Errorf("expected output to contain 'silo-abc12345 already exists', got: %s", output)
			}
			// And no build should occur
			mock.AssertNoExec("podman", "build", "<...>")
			// And the exit code should be 0 (implicit)
		})

		t.Run("Scenario: user image exists, workspace missing", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			// And the user's XDG_CONFIG_HOME points to a fresh directory
			// And the user's silo config directory has all starter files
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And the user image "silo-alice" exists
			// And no workspace image exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":    exec.Command("true"),
				"podman image exists silo-abc12345": exec.Command("false"),
				"podman container exists silo-abc12345": exec.Command("false"),
			})

			// When I run `silo build`
			output := internal.CaptureStdout(func() { cmd.Build([]string{}) })

			// Then the user image should not be rebuilt
			mock.AssertNoExec("podman", "build", "-t", "silo-alice", "<...>")
			// And the workspace image "silo-abc12345" should be built
			mock.AssertExec("podman", "build", "-t", "silo-abc12345", "<...>")
			// And the output should contain "Building workspace image silo-abc12345..."
			if !strings.Contains(output, "Building workspace image silo-abc12345...") {
				t.Errorf("expected output to contain 'Building workspace image silo-abc12345...', got: %s", output)
			}
		})
	})

	t.Run("Rule: Init on demand — build initializes workspace if not initialized", func(t *testing.T) {
		t.Run("Scenario: build creates workspace config if missing", func(t *testing.T) {
			// Given a clean workspace with no existing silo files
			// And the user's XDG_CONFIG_HOME points to a fresh directory
			// And the user's silo config directory has all starter files
			internal.FirstRunWithFiles(t, map[string]string{
				"home-user.nix": internal.HomeUserNix,
				"silo.in.toml":  "",
			})

			// Control the generated ID so we can verify exact names
			internal.SetGeneratedIDFunc(t, func() string { return "abc12345" })

			// And no user image exists
			// And no workspace image exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists <any>":        exec.Command("false"),
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
			// And the user image "silo-markus" should be built
			userBuild := mock.AssertExec("podman", "build", "-t", "silo-markus", "<...>")
			// And the workspace image "silo-abc12345" should be built
			workspaceBuild := mock.AssertExec("podman", "build", "-t", "silo-abc12345", "<...>")
			// Verify sequence: user image built before workspace image
			if userBuild != nil && workspaceBuild != nil && userBuild.Seq >= workspaceBuild.Seq {
				t.Error("expected user image to be built before workspace image")
			}
		})
	})

	t.Run("Rule: home.nix is baked into the workspace image", func(t *testing.T) {
		t.Run("Scenario: workspace home.nix content is included in the built image", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And the workspace has "home.nix" with content:
			homeNix := filepath.Join(internal.SiloDir(), "home.nix")
			expectedContent := "home.packages = with pkgs; [ nodejs python3 ];\n"
			if err := os.WriteFile(homeNix, []byte(expectedContent), 0644); err != nil {
				t.Fatalf("write home.nix: %v", err)
			}

			// And no user image exists
			// And no workspace image exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":    exec.Command("false"),
				"podman image exists silo-abc12345": exec.Command("false"),
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
			// And the workspace image build should include a file "home-workspace.nix" containing "nodejs python3"
			mock.AssertRead(homeNix)
		})
	})

	t.Run("Rule: --force forces workspace image rebuild", func(t *testing.T) {
		t.Run("Scenario: build --force rebuilds even when image exists", func(t *testing.T) {
			// Given the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			// And the container "silo-abc12345" does not exist
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":    exec.Command("true"),
				"podman image exists silo-abc12345": exec.Command("true"),
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
			// Given the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			// And the container "silo-abc12345" is running
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":    exec.Command("true"),
				"podman image exists silo-abc12345": exec.Command("true"),
				"podman container exists silo-abc12345":    exec.Command("true"),
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
			// Given the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			// And the container "silo-abc12345" exists but is stopped
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists silo-alice":    exec.Command("true"),
				"podman image exists silo-abc12345": exec.Command("true"),
				"podman container exists silo-abc12345":    exec.Command("true"),
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
}
