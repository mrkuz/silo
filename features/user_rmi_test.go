package features_test

import (
	"os/exec"
	"os/user"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

// Feature: silo user rmi — Remove the shared user image
// `silo user rmi` removes the shared user image (`silo-<username>`) for the
// current user. It has no effect if the image does not exist.
func TestFeatureUserRmi(t *testing.T) {
	// Background: the user's XDG_CONFIG_HOME points to a fresh directory
	currentUser, _ := user.Current()
	userImage := "silo-" + currentUser.Username

	t.Run("Rule: Removes the user image when present", func(t *testing.T) {
		t.Run("Scenario: user rmi removes the user image", func(t *testing.T) {
			internal.SetupUserConfig(t)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists " + userImage: exec.Command("true"),
				"podman rmi " + userImage:           exec.Command("true"),
			})

			// When I run `silo user rmi`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.UserRmi()
			})

			// Then the user image should be removed
			mock.AssertExec("podman", "rmi", userImage)
			// And the output should contain "Removing <userImage>..."
			if !strings.Contains(output, "Removing "+userImage+"...") {
				t.Errorf("expected 'Removing %s...' in output, got: %s", userImage, output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: user rmi prints removal message", func(t *testing.T) {
			internal.SetupUserConfig(t)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists " + userImage: exec.Command("true"),
				"podman rmi " + userImage:           exec.Command("true"),
			})

			// When I run `silo user rmi`
			output := internal.CaptureStdout(func() {
				cmd.UserRmi()
			})

			// Then the output should contain "Removing <userImage>..."
			if !strings.Contains(output, "Removing "+userImage+"...") {
				t.Errorf("expected 'Removing %s...' in output, got: %s", userImage, output)
			}
		})
	})

	t.Run("Rule: Idempotency — image not found is not an error", func(t *testing.T) {
		t.Run("Scenario: missing user image is a no-op", func(t *testing.T) {
			internal.SetupUserConfig(t)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists " + userImage: exec.Command("false"),
			})

			// When I run `silo user rmi`
			var err error
			output := internal.CaptureStdout(func() {
				err = cmd.UserRmi()
			})

			// Then the output should contain "<userImage> not found"
			if !strings.Contains(output, userImage+" not found") {
				t.Errorf("expected '%s not found' in output, got: %s", userImage, output)
			}
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})

		t.Run("Scenario: missing user image does not call podman rmi", func(t *testing.T) {
			internal.SetupUserConfig(t)
			mock := internal.NewMock(t)
			mock.MockExec(map[string]*exec.Cmd{
				"podman image exists " + userImage: exec.Command("false"),
			})

			// When I run `silo user rmi`
			err := cmd.UserRmi()

			// Then no podman rmi call should be made
			mock.AssertNoExec("podman", "rmi", "<any>")
			// And the exit code should be 0
			if err != nil {
				t.Errorf("expected exit code 0, got error: %v", err)
			}
		})
	})
}