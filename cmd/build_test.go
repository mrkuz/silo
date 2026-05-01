package cmd_test

import (
	"os/exec"
	"testing"

	"github.com/mrkuz/silo/internal"
)

// TestImageExists is a unit test for the internal ImageExists function.
// Covered by integration tests in features/build_test.go for the full build flow.
func TestImageExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists silo-test": exec.Command("true"),
		})
		if !internal.ImageExists("silo-test") {
			t.Error("expected ImageExists to return true")
		}
		mock.AssertExec("podman", "image", "exists", "silo-test")
	})

	t.Run("not exists", func(t *testing.T) {
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists silo-test": exec.Command("false"),
		})
		if internal.ImageExists("silo-test") {
			t.Error("expected ImageExists to return false")
		}
	})
}
