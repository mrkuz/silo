package features_test

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo user build — Build the shared user image
// `silo user build` builds the shared user image (`silo-<username>`) if it does not
// already exist. The user image is shared across all workspaces and includes the
// user's `home-user.nix`. It is a prerequisite for workspace image builds.
func TestFeatureUserBuild(t *testing.T) {
	// Background: the user's XDG_CONFIG_HOME points to a fresh directory
	// and the user's silo config directory has all starter files

	currentUser, _ := user.Current()
	userImage := "silo-" + currentUser.Username

	t.Run("Rule: Builds the user image if missing", func(t *testing.T) {
		t.Run("Scenario: missing user image triggers build", func(t *testing.T) {
			internal.SetupUserConfig(t)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists " + userImage:        exec.Command("false"),
				"podman build -t " + userImage + " <...>": exec.Command("true"),
			})

			// When I run `silo user build`
			err := cmd.UserBuild()

			// Then the user image should be built
			mock.AssertExec("podman", "build", "-t", userImage, "<...>")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: build prints a message while building", func(t *testing.T) {
			internal.SetupUserConfig(t)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists " + userImage:        exec.Command("false"),
				"podman build -t " + userImage + " <...>": exec.Command("true"),
			})

			// When I run `silo user build`
			output := internal.CaptureStdout(func() {
				cmd.UserBuild()
			})

			// Then the output should contain "Building user image <userImage>..."
			if !strings.Contains(output, "Building user image "+userImage+"...") {
				t.Errorf("expected 'Building user image %s...' in output, got: %s", userImage, output)
			}
		})
	})

	t.Run("Rule: Idempotency — existing image is not rebuilt", func(t *testing.T) {
		t.Run("Scenario: existing user image is skipped", func(t *testing.T) {
			internal.SetupUserConfig(t)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists " + userImage: exec.Command("true"),
			})

			// When I run `silo user build`
			output := internal.CaptureStdout(func() {
				cmd.UserBuild()
			})

			// Then the output should contain "<userImage> already exists"
			if !strings.Contains(output, userImage+" already exists") {
				t.Errorf("expected '%s already exists' in output, got: %s", userImage, output)
			}
			// And no build should occur
			mock.AssertNoExec("podman", "build", "<any>")
		})
	})

	t.Run("Rule: Automatically runs user init if needed", func(t *testing.T) {
		t.Run("Scenario: missing user files triggers automatic user init", func(t *testing.T) {
			// Given a fresh XDG_CONFIG_HOME without starter files
			base := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", base)
			// And no user image exists
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists " + userImage:        exec.Command("false"),
				"podman build -t " + userImage + " <...>": exec.Command("true"),
			})

			// When I run `silo user build`
			err := cmd.UserBuild()

			// Then the user files should be created
			// And the user image should be built
			mock.AssertExec("podman", "build", "-t", userImage, "<...>")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: existing user files are preserved during auto init", func(t *testing.T) {
			internal.SetupUserConfig(t)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists " + userImage:        exec.Command("false"),
				"podman build -t " + userImage + " <...>": exec.Command("true"),
			})

			// When I run `silo user build`
			err := cmd.UserBuild()

			// Then the user files should not be modified
			// And the user image should be built
			mock.AssertExec("podman", "build", "-t", userImage, "<...>")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})

	t.Run("Rule: home-user.nix is baked into the user image", func(t *testing.T) {
		t.Run("Scenario: user's home-user.nix content is included in the built image", func(t *testing.T) {
			internal.SetupUserConfig(t)
			mock := internal.NewMock(t)

			xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
			homeUserNix := filepath.Join(xdgConfigHome, "silo", "home-user.nix")
			expectedContent := "{ config, pkgs, ... }:\n{\n}\n"

			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists " + userImage:        exec.Command("false"),
				"podman build -t " + userImage + " <...>": exec.Command("true"),
			})
			mock.MockRead(map[string][]byte{
				homeUserNix: []byte(expectedContent),
			})

			// When I run `silo user build`
			if err := cmd.UserBuild(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then the podman build command should be called
			mock.AssertExec("podman", "build", "-t", userImage, "<...>")
			// And the build context should include home-user.nix
			mock.AssertRead(homeUserNix)
		})
	})
}
