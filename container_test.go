package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestSecurityArgsNested(t *testing.T) {
	args := securityArgs(true)
	if !strings.Contains(strings.Join(args, " "), "label=disable") {
		t.Errorf("nested mode should include label=disable, got %v", args)
	}
	if !strings.Contains(strings.Join(args, " "), "/dev/fuse") {
		t.Errorf("nested mode should include /dev/fuse, got %v", args)
	}
}

func TestSecurityArgsNonNested(t *testing.T) {
	args := securityArgs(false)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--cap-drop=ALL") {
		t.Errorf("non-nested mode should include --cap-drop=ALL, got %v", args)
	}
	if !strings.Contains(joined, "no-new-privileges") {
		t.Errorf("non-nested mode should include no-new-privileges, got %v", args)
	}
	if strings.Contains(joined, "label=disable") {
		t.Errorf("non-nested mode should not include label=disable, got %v", args)
	}
}

func TestBuildContainerArgsMinimal(t *testing.T) {
	cfg := Config{
		General: GeneralConfig{
			ID:            "abc12345",
			User:          "alice",
			ContainerName: "silo-abc12345",
			ImageName:     "silo-abc12345",
		},
		Features: FeaturesConfig{
			Workspace:    false,
			SharedVolume: false,
			Nested:       false,
		},
	}
	args, err := buildContainerArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--name silo-abc12345") {
		t.Errorf("expected --name silo-abc12345 in args: %v", args)
	}
	if !strings.Contains(joined, "--hostname silo-abc12345") {
		t.Errorf("expected --hostname silo-abc12345 in args: %v", args)
	}
	if !strings.Contains(joined, "--user alice") {
		t.Errorf("expected --user alice in args: %v", args)
	}
	if strings.Contains(joined, "--volume") {
		t.Errorf("expected no --volume without workspace or shared volume, got: %v", args)
	}
}

func TestBuildContainerArgsSharedVolume(t *testing.T) {
	cfg := Config{
		General: GeneralConfig{
			ID:            "abc12345",
			User:          "alice",
			ContainerName: "silo-abc12345",
			ImageName:     "silo-abc12345",
		},
		Features: FeaturesConfig{
			Workspace:    false,
			SharedVolume: true,
			Nested:       false,
		},
	}
	args, err := buildContainerArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, sharedVolume+":/silo/shared:Z") {
		t.Errorf("expected shared volume mount in args: %v", args)
	}
}

func TestBuildContainerArgsWorkspace(t *testing.T) {
	cfg := Config{
		General: GeneralConfig{
			ID:            "abc12345",
			User:          "alice",
			ContainerName: "silo-abc12345",
			ImageName:     "silo-abc12345",
		},
		Features: FeaturesConfig{
			Workspace:    true,
			SharedVolume: false,
			Nested:       false,
		},
	}
	args, err := buildContainerArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "/workspace/abc12345/") {
		t.Errorf("expected workspace mount path in args: %v", args)
	}
	if !strings.Contains(joined, "--workdir") {
		t.Errorf("expected --workdir in args: %v", args)
	}
}

func TestBuildContainerArgsNested(t *testing.T) {
	cfg := Config{
		General: GeneralConfig{
			ID:            "abc12345",
			User:          "alice",
			ContainerName: "silo-abc12345",
			ImageName:     "silo-abc12345",
		},
		Features: FeaturesConfig{
			Workspace:    false,
			SharedVolume: false,
			Nested:       true,
		},
	}
	args, err := buildContainerArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "label=disable") {
		t.Errorf("expected nested security args, got: %v", args)
	}
}

// ---- setupContainer tests ------------------------------------------------

func TestSetupContainer(t *testing.T) {
	t.Run("skipped when shared volume disabled", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.SharedVolume = false
		calls := mockExecCommand(t, map[string]*exec.Cmd{})
		if err := setupContainer(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "exec") {
			t.Errorf("expected no podman exec, got %v", *calls)
		}
	})

	t.Run("skipped when paths empty", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{}
		calls := mockExecCommand(t, map[string]*exec.Cmd{})
		if err := setupContainer(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "exec") {
			t.Errorf("expected no podman exec, got %v", *calls)
		}
	})

	t.Run("runs startup script when paths configured", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
		calls := mockExecCommand(t, map[string]*exec.Cmd{})
		_ = setupContainer(cfg)
		if !anyCall(calls, "podman", "exec", "silo-abc12345", "bash", "/silo/setup.sh") {
			t.Errorf("expected podman exec silo-abc12345 bash /silo/setup.sh, got %v", *calls)
		}
	})

}

// ---- helper-function tests -----------------------------------------------

func TestContainerExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc": exec.Command("true"),
		})
		if !containerExists("silo-abc") {
			t.Error("expected containerExists to return true")
		}
	})

	t.Run("not exists", func(t *testing.T) {
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc": exec.Command("false"),
		})
		if containerExists("silo-abc") {
			t.Error("expected containerExists to return false")
		}
	})
}

func TestContainerRunning(t *testing.T) {
	t.Run("running", func(t *testing.T) {
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc": exec.Command("echo", "true"),
		})
		if !containerRunning("silo-abc") {
			t.Error("expected containerRunning to return true")
		}
	})

	t.Run("not running", func(t *testing.T) {
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc": exec.Command("echo", "false"),
		})
		if containerRunning("silo-abc") {
			t.Error("expected containerRunning to return false")
		}
	})

	t.Run("podman error", func(t *testing.T) {
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc": exec.Command("false"),
		})
		if containerRunning("silo-abc") {
			t.Error("expected containerRunning false on error")
		}
	})
}

func TestEnsureChain(t *testing.T) {
	t.Run("container absent — creates, starts, runs setup", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
		setupWorkspace(t, cfg)
		setupGlobalConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                exec.Command("true"),
			"podman image exists silo-abc12345":                                exec.Command("true"),
			"podman container exists silo-abc12345":                            exec.Command("false"),
					})
		_, err := ensureSetup()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "create") {
			t.Errorf("expected podman create, got %v", *calls)
		}
		if !anyCall(calls, "podman", "start", "silo-abc12345") {
			t.Errorf("expected podman start, got %v", *calls)
		}
		if !anyCall(calls, "podman", "exec", "silo-abc12345", "bash", "/silo/setup.sh") {
			t.Errorf("expected setup via podman exec, got %v", *calls)
		}
	})

	t.Run("container stopped — starts and runs setup", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
		setupWorkspace(t, cfg)
		setupGlobalConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
			"podman exec silo-abc12345 test -f /silo/.setup-done":               exec.Command("false"),
		})
		_, err := ensureSetup()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "start", "silo-abc12345") {
			t.Errorf("expected podman start, got %v", *calls)
		}
		if !anyCall(calls, "podman", "exec", "silo-abc12345", "bash", "/silo/setup.sh") {
			t.Errorf("expected setup via podman exec, got %v", *calls)
		}
	})

	t.Run("container already running — runs setup only", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupGlobalConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		_, err := ensureSetup()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "create") || anyCall(calls, "podman", "start") {
			t.Errorf("expected no create or start, got %v", *calls)
		}
	})
}

func TestCreateContainerExtraArgs(t *testing.T) {
	cfg := minimalConfig("abc12345")
	cfg.Create.ExtraArgs = []string{"--memory", "512m"}
	calls := mockExecCommand(t, map[string]*exec.Cmd{})
	_ = createContainer(cfg, cfg.Create.ExtraArgs)
	if !anyCall(calls, "podman", "create", "--memory", "512m") {
		t.Errorf("expected --memory 512m in podman create call, got %v", *calls)
	}
}
