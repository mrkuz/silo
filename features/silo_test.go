package features_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo (default invocation) — Run lifecycle and connect to the container
// The default `silo` invocation (no subcommand) runs the full lifecycle chain
// (init → build → start) if needed, then opens an interactive shell session
// inside the running container. After the session exits, cleanup flags control what
// is stopped or removed.
func TestFeatureSilo(t *testing.T) {
	// Background: a workspace with silo config "abc12345"
	// and the user's XDG_CONFIG_HOME points to a fresh directory

	t.Run("Rule: Connects to the running container", func(t *testing.T) {
		t.Run("Scenario: default silo connects to the container", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			// And the user's XDG_CONFIG_HOME points to a fresh directory
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And the container "silo-abc12345" is running
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
			})

			// When I run `silo`
			err := cmd.Run([]string{})

			// Then podman should run "exec" with "-ti" on "silo-abc12345"
			mock.AssertExec("podman", "exec", "-ti", "silo-abc12345", "<...>")
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: without cleanup flags, container keeps running after session ends", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And the container "silo-abc12345" is running
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
			})

			// When I run `silo`
			err := cmd.Run([]string{})

			// And the interactive session ends
			// Then the container "silo-abc12345" should still be running
			// But podman should not run "stop" on "silo-abc12345"
			mock.AssertNoExec("podman", "stop", "<any>")
			// And podman should not run "rm" on "silo-abc12345"
			mock.AssertNoExec("podman", "rm", "<any>")
			// And podman should not run "rmi" on "silo-abc12345"
			mock.AssertNoExec("podman", "rmi", "<any>")
			// And the command should be "/bin/sh"
			execRecord := mock.AssertExec("podman", "exec", "<any>", "silo-abc12345", "/bin/sh")
			if execRecord == nil {
				t.Error("expected exec to be called")
			}
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: extra args after -- are passed to podman exec", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And the container "silo-abc12345" is running
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
			})

			// When I run `silo -- -p 8080:8080`
			err := cmd.Run([]string{"--", "-p", "8080:8080"})

			// Then podman should run "exec" with "-ti" on "silo-abc12345" with extra args "--" "-p" "8080:8080"
			execCall := mock.AssertExec("podman", "exec", "-ti", "<...>", "silo-abc12345", "<...>")
			if execCall != nil {
				argsStr := strings.Join(execCall.Args, " ")
				if !strings.Contains(argsStr, "-p") || !strings.Contains(argsStr, "8080:8080") {
					t.Errorf("expected exec to include port mapping, got: %s", argsStr)
				}
			}
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: --stop stops the container after the session exits", func(t *testing.T) {
		t.Run("Scenario: container is stopped after shell exits", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And the container "silo-abc12345" is running
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
			})

			// When I run `silo --stop`
			err := cmd.Run([]string{"--stop"})

			// And the interactive session ends
			// Then podman should run "stop" with "-t" and "0" on "silo-abc12345"
			mock.AssertExec("podman", "stop", "-t", "0", "silo-abc12345")
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: --stop removes the container", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And the container "silo-abc12345" is running
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
			})

			// When I run `silo --stop`
			err := cmd.Run([]string{"--stop"})

			// And the interactive session ends
			// Then podman should run "stop" on "silo-abc12345"
			mock.AssertExec("podman", "stop", "-t", "0", "silo-abc12345")
			// And podman should run "rm" on "silo-abc12345"
			mock.AssertExec("podman", "rm", "-f", "silo-abc12345")
			// But podman should not run "rmi" on "silo-abc12345"
			mock.AssertNoExec("podman", "rmi", "<any>")
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: --rm stops, removes container, and removes image after the session exits", func(t *testing.T) {
		t.Run("Scenario: container and image are removed", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And the container "silo-abc12345" is running
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
			})

			// When I run `silo --rm`
			err := cmd.Run([]string{"--rm"})

			// And the interactive session ends
			// Then podman should run "stop" with "-t" and "0" on "silo-abc12345"
			mock.AssertExec("podman", "stop", "-t", "0", "silo-abc12345")
			// And podman should run "rm" with "-f" on "silo-abc12345"
			mock.AssertExec("podman", "rm", "-f", "silo-abc12345")
			// And podman should run "rmi" on "silo-abc12345"
			mock.AssertExec("podman", "rmi", "silo-abc12345")
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: Runs the full lifecycle chain if needed", func(t *testing.T) {
		t.Run("Scenario: stopped container triggers start", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And the container "silo-abc12345" exists but is stopped
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
			})

			// When I run `silo`
			err := cmd.Run([]string{})

			// Then podman should run "start" on "silo-abc12345"
			mock.AssertExec("podman", "start", "silo-abc12345")
			// And podman should run "exec" with "-ti" on "silo-abc12345"
			mock.AssertExec("podman", "exec", "-ti", "silo-abc12345", "<...>")
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: missing container triggers full build-and-create chain", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And no container exists
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("true"),
			})

			// When I run `silo`
			err := cmd.Run([]string{})

			// Then the container "silo-abc12345" should be created
			mock.AssertExec("podman", "create", "<...>")
			// And podman should run "start" on "silo-abc12345"
			mock.AssertExec("podman", "start", "silo-abc12345")
			// And podman should run "exec" with "-ti" on "silo-abc12345"
			mock.AssertExec("podman", "exec", "-ti", "silo-abc12345", "<...>")
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: fresh workspace triggers full lifecycle: init, user image build, workspace image build, create, volume setup, start, connect", func(t *testing.T) {
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
			// And no container exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists <any>": exec.Command("false"),
				"podman image exists <any>":     exec.Command("false"),
			})

			// When I run `silo`
			err := cmd.Run([]string{})

			// Then workspace files should be created: ".silo/silo.toml" and ".silo/home.nix"
			if _, statErr := os.Stat(internal.SiloToml()); statErr != nil {
				t.Errorf("expected .silo/silo.toml to be created: %v", statErr)
			}
			if _, statErr := os.Stat(internal.SiloDir() + "/home.nix"); statErr != nil {
				t.Errorf("expected .silo/home.nix to be created: %v", statErr)
			}
			// And a user image should be built (silo-markus)
			userBuild := mock.AssertExec("podman", "build", "-t", "silo-markus", "<...>")
			// And a workspace image "silo-abc12345" should be built
			workspaceBuild := mock.AssertExec("podman", "build", "-t", "silo-abc12345", "<...>")
			// And the container should be created
			create := mock.AssertExec("podman", "create", "<...>")
			// And podman should run "start"
			start := mock.AssertExec("podman", "start", "silo-abc12345")
			// And podman should run "exec" with "-ti"
			exec := mock.AssertExec("podman", "exec", "-ti", "silo-abc12345", "<...>")

			// Verify full sequence ordering
			if userBuild.Seq >= workspaceBuild.Seq {
				t.Error("expected user image build before workspace image build")
			}
			if workspaceBuild.Seq >= create.Seq {
				t.Error("expected workspace image build before container create")
			}
			if create.Seq >= start.Seq {
				t.Error("expected container create before container start")
			}
			if start.Seq >= exec.Seq {
				t.Error("expected container start before exec")
			}
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: missing user image triggers user image build first", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And no user image exists
			// And no workspace image exists
			// And no container exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman image exists silo-alice":        exec.Command("false"),
				"podman image exists silo-abc12345":     exec.Command("false"),
			})

			// When I run `silo`
			err := cmd.Run([]string{})

			// Then the user image "silo-alice" should be built
			userBuild := mock.AssertExec("podman", "build", "-t", "silo-alice", "<...>")
			// And the workspace image "silo-abc12345" should be built
			workspaceBuild := mock.AssertExec("podman", "build", "-t", "silo-abc12345", "<...>")
			// And the container should be created
			mock.AssertExec("podman", "create", "<...>")
			// And podman should run "exec" with "-ti"
			mock.AssertExec("podman", "exec", "-ti", "<any>", "<...>")
			// User image should be built before workspace image
			if userBuild != nil && workspaceBuild != nil && userBuild.Seq >= workspaceBuild.Seq {
				t.Error("expected user image to be built before workspace image")
			}
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: missing workspace image triggers workspace image build", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			internal.SubsequentRun(t, cfg)

			// And the user image "silo-alice" exists
			// And no workspace image exists
			// And no container exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345": exec.Command("false"),
				"podman image exists silo-alice":        exec.Command("true"),
				"podman image exists silo-abc12345":     exec.Command("false"),
			})

			// When I run `silo`
			err := cmd.Run([]string{})

			// Then the workspace image "silo-abc12345" should be built
			mock.AssertExec("podman", "build", "-t", "silo-abc12345", "<...>")
			// And the container "silo-abc12345" should be created
			mock.AssertExec("podman", "create", "<...>")
			// And podman should run "exec" with "-ti" on "silo-abc12345"
			mock.AssertExec("podman", "exec", "-ti", "silo-abc12345", "<...>")
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: volume setup runs before container start when shared volume is configured", func(t *testing.T) {
			// Given a workspace with silo config "abc12345"
			cfg := internal.MinimalConfig("abc12345")
			cfg.General.User = "alice"
			cfg.Features.SharedVolume = true
			cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
			internal.SubsequentRun(t, cfg)

			// And the container "silo-abc12345" exists but is stopped
			// And the user image "silo-alice" exists
			// And the workspace image "silo-abc12345" exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman container exists silo-abc12345":                              exec.Command("true"),
				"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
				"podman image exists silo-alice":                                     exec.Command("true"),
				"podman image exists silo-abc12345":                                  exec.Command("true"),
			})

			// When I run `silo`
			err := cmd.Run([]string{})

			// Then volume setup should run before start, creating directories on the shared volume
			volumeSetup := mock.AssertExec("podman", "run", "--rm", "<...>")
			cmdStr := strings.Join(volumeSetup.Args, " ")
			// Verify volume setup creates the expected directory path
			expectedPath := "/silo/shared/home/alice/.cache/uv"
			if !strings.Contains(cmdStr, "mkdir -p "+expectedPath) {
				t.Errorf("expected mkdir -p %s, got: %s", expectedPath, cmdStr)
			}
			start := mock.AssertExec("podman", "start", "silo-abc12345")
			// Verify the sequence: volumeSetup should come before start
			if volumeSetup.Seq >= start.Seq {
				t.Error("expected volume setup to complete before container start")
			}
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})
}
