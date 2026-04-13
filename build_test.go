package main

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"testing"
)

func TestCmdUserBuildSharedVolumeMount(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("get current user: %v", err)
	}
	cfg := Config{
		General: GeneralConfig{
			User:          u.Username,
			ContainerName: "silo-" + u.Username,
		},
		Features: FeaturesConfig{
			SharedVolume: true,
		},
		SharedVolume: SharedVolumeConfig{
			Name:  "silo-shared",
			Paths: []string{"$HOME/.cache/uv/"},
		},
	}
	tc, err := newTemplateContext(cfg)
	if err != nil {
		t.Fatalf("newTemplateContext failed: %v", err)
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
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-test": exec.Command("true"),
		})
		if !imageExists("silo-test") {
			t.Error("expected imageExists to return true")
		}
		if !anyCall(calls, "podman", "image", "exists", "silo-test") {
			t.Errorf("expected podman image exists call, got %v", *calls)
		}
	})

	t.Run("not exists", func(t *testing.T) {
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-test": exec.Command("false"),
		})
		if imageExists("silo-test") {
			t.Error("expected imageExists to return false")
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
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists " + userImage: exec.Command("true"),
		})

		if err := cmdUserBuild(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if anyCall(calls, "podman", "build", "-t", userImage) {
			t.Errorf("expected no podman build for existing user image, got %v", *calls)
		}
	})

	t.Run("missing user image: builds user image", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists " + userImage: exec.Command("false"),
		})

		if err := cmdUserBuild(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if anyCall(calls, "podman", "build", "-t", "silo-abc12345") {
			t.Errorf("expected no workspace image build, got %v", *calls)
		}
		if !anyCall(calls, "podman", "build", "-t", userImage) {
			t.Errorf("expected podman build -t %s, got %v", userImage, *calls)
		}
	})
}

func TestCmdBuild(t *testing.T) {
	t.Run("existing workspace image: no-op", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-abc12345": exec.Command("true"),
		})

		if err := cmdBuild(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if anyCall(calls, "podman", "rmi", "silo-abc12345") {
			t.Errorf("expected no podman rmi for workspace image, got %v", *calls)
		}
		if anyCall(calls, "podman", "build", "-t", "silo-abc12345") {
			t.Errorf("expected no podman build for existing workspace image, got %v", *calls)
		}
		if anyCall(calls, "podman", "build", "-t", "silo-testuser") {
			t.Errorf("expected no user image build, got %v", *calls)
		}
	})

	t.Run("missing workspace image: builds workspace image", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-abc12345": exec.Command("false"),
		})

		if err := cmdBuild(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !anyCall(calls, "podman", "build", "-t", "silo-abc12345") {
			t.Errorf("expected podman build -t silo-abc12345, got %v", *calls)
		}
		if anyCall(calls, "podman", "build", "-t", "silo-testuser") {
			t.Errorf("expected no user image build, got %v", *calls)
		}
	})

	t.Run("missing user and workspace images: builds user image first", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser": exec.Command("false"),
			"podman image exists silo-abc12345": exec.Command("false"),
		})

		if err := cmdBuild(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !anyCall(calls, "podman", "build", "-t", "silo-testuser") {
			t.Errorf("expected podman build -t silo-testuser, got %v", *calls)
		}
		if !anyCall(calls, "podman", "build", "-t", "silo-abc12345") {
			t.Errorf("expected podman build -t silo-abc12345, got %v", *calls)
		}
	})
}

func TestBuildWorkspaceImageWithMissingHomeNix(t *testing.T) {
	t.Run("builds workspace image when home.nix is absent", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		// Remove home.nix to simulate missing file
		os.Remove(filepath.Join(siloDir, "home.nix"))
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser": exec.Command("false"),
			"podman image exists silo-abc12345": exec.Command("false"),
		})
		if err := cmdBuild(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "build", "-t", "silo-abc12345") {
			t.Errorf("expected workspace image build even without home.nix, got %v", *calls)
		}
	})
}

// TestEnsureImagesBuildFailure and TestEnsureBuiltFailure are not feasible to test
// because runBuild uses os.MkdirTemp producing dynamic build directory paths.
// The mock cannot match the full command string with dynamic paths.
