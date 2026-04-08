package main

import (
	"os/exec"
	"testing"
)

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
	t.Run("existing user image: no-op", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser": exec.Command("true"),
		})

		if err := cmdUserBuild(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if anyCall(calls, "podman", "build", "-t", "silo-testuser") {
			t.Errorf("expected no podman build for existing user image, got %v", *calls)
		}
	})

	t.Run("missing user image: builds user image", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser": exec.Command("false"),
		})

		if err := cmdUserBuild(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if anyCall(calls, "podman", "build", "-t", "silo-abc12345") {
			t.Errorf("expected no workspace image build, got %v", *calls)
		}
		if !anyCall(calls, "podman", "build", "-t", "silo-testuser") {
			t.Errorf("expected podman build -t silo-testuser, got %v", *calls)
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
