package cmd_test

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

func TestCmdUserBuildSharedVolumeMount(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("get current user: %v", err)
	}
	cfg := internal.Config{
		General: internal.GeneralConfig{
			User:          u.Username,
			ContainerName: "silo-" + u.Username,
		},
		Features: internal.FeaturesConfig{
			SharedVolume: true,
		},
		SharedVolume: internal.SharedVolumeConfig{
			Name:  "silo-shared",
			Paths: []string{"$HOME/.cache/uv/"},
		},
	}
	tc, err := internal.NewTemplateContext(cfg)
	if err != nil {
		t.Fatalf("NewTemplateContext failed: %v", err)
	}
	if tc.SharedVolumeName == "" {
		t.Error("SharedVolumeName should not be empty when SharedVolume is enabled")
	}
	if len(tc.SharedVolumePaths) == 0 {
		t.Error("SharedVolumePaths should not be empty when paths are configured")
	}
}

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

func TestCmdUserBuild(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("get current user: %v", err)
	}
	userImage := "silo-" + u.Username

	t.Run("existing user image: no-op", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		internal.SetupUserConfig(t)
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists " + userImage: exec.Command("true"),
		})

		if err := cmd.UserBuild(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mock.AssertNoExec("podman", "build", "-t", userImage, "<...>")
	})

	t.Run("missing user image: builds user image", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		internal.SetupUserConfig(t)
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists " + userImage: exec.Command("false"),
		})

		if err := cmd.UserBuild(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mock.AssertNoExec("podman", "build", "-t", "silo-abc12345", "<...>")
		mock.AssertExec("podman", "build", "-t", userImage, "<...>")
	})
}

func TestCmdBuild(t *testing.T) {
	t.Run("existing workspace image: no-op", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		internal.SetupUserConfig(t)
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists silo-abc12345": exec.Command("true"),
		})

		if err := cmd.Build(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mock.AssertNoExec("podman", "rmi", "silo-abc12345")
		mock.AssertNoExec("podman", "build", "-t", "silo-abc12345", "<...>")
		mock.AssertNoExec("podman", "build", "-t", "silo-testuser", "<...>")
	})

	t.Run("missing workspace image: builds workspace image", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		internal.SetupUserConfig(t)
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists silo-abc12345": exec.Command("false"),
		})

		if err := cmd.Build(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mock.AssertExec("podman", "build", "-t", "silo-abc12345", "<...>")
		mock.AssertNoExec("podman", "build", "-t", "silo-testuser", "<...>")
	})

	t.Run("missing user and workspace images: builds user image first", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		internal.SetupUserConfig(t)
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists silo-testuser": exec.Command("false"),
			"podman image exists silo-abc12345": exec.Command("false"),
		})

		if err := cmd.Build(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mock.AssertExec("podman", "build", "-t", "silo-testuser", "<...>")
		mock.AssertExec("podman", "build", "-t", "silo-abc12345", "<...>")
	})
}

func TestBuildWorkspaceImageWithMissingHomeNix(t *testing.T) {
	t.Run("builds workspace image when home.nix is absent", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		internal.SetupUserConfig(t)
		// Remove home.nix to simulate missing file
		os.Remove(filepath.Join(internal.SiloDir(), "home.nix"))
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists silo-testuser": exec.Command("false"),
			"podman image exists silo-abc12345": exec.Command("false"),
		})
		if err := cmd.Build(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mock.AssertExec("podman", "build", "-t", "silo-abc12345", "<...>")
	})
}
