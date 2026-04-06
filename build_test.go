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

func TestParseBuildFlags(t *testing.T) {
	tests := []struct {
		args     []string
		wantUser bool
		wantErr  bool
	}{
		{[]string{}, false, false},
		{[]string{"--user"}, true, false},
		{[]string{"--unknown"}, false, true},
	}
	for _, tt := range tests {
		f, err := parseBuildFlags(tt.args)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseBuildFlags(%v): expected error", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseBuildFlags(%v): unexpected error: %v", tt.args, err)
			continue
		}
		if f.user != tt.wantUser {
			t.Errorf("parseBuildFlags(%v).user = %v, want %v", tt.args, f.user, tt.wantUser)
		}
	}
}

func TestCmdBuild(t *testing.T) {
	t.Run("--user with existing user image: no-op", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser": exec.Command("true"),
		})

		if err := cmdBuild([]string{"--user"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if anyCall(calls, "podman", "rmi", "silo-testuser") {
			t.Errorf("expected no podman rmi for user image, got %v", *calls)
		}
		if anyCall(calls, "podman", "build", "-t", "silo-testuser") {
			t.Errorf("expected no podman build for existing user image, got %v", *calls)
		}
	})

	t.Run("--user with missing user image: builds user image", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser": exec.Command("false"),
		})

		if err := cmdBuild([]string{"--user"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if anyCall(calls, "podman", "build", "-t", "silo-abc12345") {
			t.Errorf("expected no workspace image build, got %v", *calls)
		}
		if !anyCall(calls, "podman", "build", "-t", "silo-testuser") {
			t.Errorf("expected podman build -t silo-testuser, got %v", *calls)
		}
	})

	t.Run("default with existing workspace image: no-op", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-abc12345": exec.Command("true"),
		})

		if err := cmdBuild([]string{}); err != nil {
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

	t.Run("default with missing workspace image: builds workspace image", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-abc12345": exec.Command("false"),
		})

		if err := cmdBuild([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !anyCall(calls, "podman", "build", "-t", "silo-abc12345") {
			t.Errorf("expected podman build -t silo-abc12345, got %v", *calls)
		}
		if anyCall(calls, "podman", "build", "-t", "silo-testuser") {
			t.Errorf("expected no user image build, got %v", *calls)
		}
	})
}
