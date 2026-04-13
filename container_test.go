package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestContainerArgsBasic(t *testing.T) {
	cfg := Config{
		General:  GeneralConfig{ContainerName: "silo-abc12345", User: "testuser"},
		Features: FeaturesConfig{Podman: true},
	}
	args := containerArgs(cfg)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--name silo-abc12345") {
		t.Errorf("expected --name silo-abc12345 in args: %v", args)
	}
	if !strings.Contains(joined, "--hostname silo-abc12345") {
		t.Errorf("expected --hostname silo-abc12345 in args: %v", args)
	}
	if !strings.Contains(joined, "--user testuser") {
		t.Errorf("expected --user testuser in args: %v", args)
	}
	// Security args are no longer in containerArgs — they live in Create.Arguments
	if strings.Contains(joined, "label=disable") || strings.Contains(joined, "/dev/fuse") {
		t.Errorf("security args should not be in containerArgs, got %v", args)
	}
}

func TestContainerArgsNonNested(t *testing.T) {
	cfg := Config{
		General:  GeneralConfig{ContainerName: "silo-abc12345", User: "testuser"},
		Features: FeaturesConfig{Podman: false},
	}
	args := containerArgs(cfg)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--name silo-abc12345") {
		t.Errorf("expected --name silo-abc12345 in args: %v", args)
	}
	if !strings.Contains(joined, "--hostname silo-abc12345") {
		t.Errorf("expected --hostname silo-abc12345 in args: %v", args)
	}
	// Security args are no longer in containerArgs — they live in Create.Arguments
	if strings.Contains(joined, "--cap-drop") || strings.Contains(joined, "no-new-privileges") {
		t.Errorf("security args should not be in containerArgs, got %v", args)
	}
}

func TestContainerArgsNameSuffix(t *testing.T) {
	cfg := Config{
		General:  GeneralConfig{ContainerName: "silo-abc12345"},
		Features: FeaturesConfig{Podman: false},
	}
	args := containerArgs(cfg, "-dev")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--name silo-abc12345-dev") {
		t.Errorf("expected --name with suffix in args: %v", args)
	}
	if !strings.Contains(joined, "--hostname silo-abc12345-dev") {
		t.Errorf("expected --hostname with suffix in args: %v", args)
	}
}

func TestWorkspaceMountPath(t *testing.T) {
	cfg := Config{
		General: GeneralConfig{ID: "abc12345"},
	}
	got, err := workspaceMountPath(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cwd, _ := os.Getwd()
	want := "/workspace/abc12345/" + filepath.Base(cwd)
	if got != want {
		t.Errorf("workspaceMountPath() = %q, want %q", got, want)
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
			SharedVolume: false,
			Podman:       false,
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
	if !strings.Contains(joined, "/workspace/abc12345/") {
		t.Errorf("expected workspace mount in args: %v", args)
	}
}

func TestBuildContainerArgsNoDuplicateFlags(t *testing.T) {
	cfg := Config{
		General: GeneralConfig{
			ID:            "abc12345",
			User:          "alice",
			ContainerName: "silo-abc12345",
			ImageName:     "silo-abc12345",
		},
		Features: FeaturesConfig{
			SharedVolume: true,
			Podman:       false,
		},
	}
	args, err := buildContainerArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Check for duplicate flag-value pairs (e.g., --user alice --user alice)
	seen := make(map[string]bool)
	for i := 0; i < len(args)-1; i++ {
		if strings.HasPrefix(args[i], "--") {
			pair := args[i] + " " + args[i+1]
			if seen[pair] {
				t.Errorf("duplicate flag-value pair %s in args: %v", pair, args)
			}
			seen[pair] = true
		}
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
			SharedVolume: true,
			Podman:       false,
		},
		SharedVolume: SharedVolumeConfig{
			Name:  "silo-shared",
			Paths: []string{"$HOME/.cache/uv/"},
		},
	}
	args, err := buildContainerArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--mount type=volume,source=silo-shared,target=/home/alice/.cache/uv,subpath=home/alice/.cache/uv,Z") {
		t.Errorf("expected subpath volume mount in args: %v", args)
	}
}

// ---- Create.Arguments tests ------------------------------------------------

func TestCreateContainerArguments(t *testing.T) {
	cfg := minimalConfig("abc12345")
	cfg.Create.Arguments = []string{"--memory", "512m"}
	calls := mockExecCommand(t, map[string]*exec.Cmd{})
	_ = createContainer(cfg, cfg.Create.Arguments)
	if !anyCall(calls, "podman", "create", "--memory", "512m") {
		t.Errorf("expected --memory 512m in podman create call, got %v", *calls)
	}
}

func TestCreateContainerArgumentsNested(t *testing.T) {
	cfg := minimalConfig("abc12345")
	cfg.Create.Arguments = []string{"--security-opt", "label=disable", "--device", "/dev/fuse"}
	calls := mockExecCommand(t, map[string]*exec.Cmd{})
	_ = createContainer(cfg, cfg.Create.Arguments)
	if !anyCall(calls, "podman", "create", "--security-opt", "label=disable") {
		t.Errorf("expected --security-opt label=disable in podman create call, got %v", *calls)
	}
	if !anyCall(calls, "podman", "create", "--device", "/dev/fuse") {
		t.Errorf("expected --device /dev/fuse in podman create call, got %v", *calls)
	}
}

func TestCreateContainerArgumentsNonNested(t *testing.T) {
	cfg := minimalConfig("abc12345")
	cfg.Create.Arguments = []string{"--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges"}
	calls := mockExecCommand(t, map[string]*exec.Cmd{})
	_ = createContainer(cfg, cfg.Create.Arguments)
	if !anyCall(calls, "podman", "create", "--cap-drop=ALL") {
		t.Errorf("expected --cap-drop=ALL in podman create call, got %v", *calls)
	}
	if !anyCall(calls, "podman", "create", "--cap-add=NET_BIND_SERVICE") {
		t.Errorf("expected --cap-add=NET_BIND_SERVICE in podman create call, got %v", *calls)
	}
	if !anyCall(calls, "podman", "create", "--security-opt", "no-new-privileges") {
		t.Errorf("expected --security-opt no-new-privileges in podman create call, got %v", *calls)
	}
}

func TestVolumeSetup(t *testing.T) {
	t.Run("skipped when shared volume disabled", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.SharedVolume = false
		calls := mockExecCommand(t, map[string]*exec.Cmd{})
		if err := volumeSetup(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "run") {
			t.Errorf("expected no podman run, got %v", *calls)
		}
	})

	t.Run("skipped when paths empty", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{}
		calls := mockExecCommand(t, map[string]*exec.Cmd{})
		if err := volumeSetup(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "run") {
			t.Errorf("expected no podman run, got %v", *calls)
		}
	})

	t.Run("runs user image to create directories", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
		calls := mockExecCommand(t, map[string]*exec.Cmd{})
		_ = volumeSetup(cfg)
		if !anyCall(calls, "podman", "run", "--rm", "-v", "silo-shared:/silo/shared:Z", "silo-testuser", "sh", "-c") {
			t.Errorf("expected podman run for volume setup, got %v", *calls)
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
	t.Run("container absent — creates and starts", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":     exec.Command("true"),
			"podman image exists silo-abc12345":     exec.Command("true"),
			"podman container exists silo-abc12345": exec.Command("false"),
		})
		_, err := ensureStarted()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "create") {
			t.Errorf("expected podman create, got %v", *calls)
		}
		if !anyCall(calls, "podman", "start", "silo-abc12345") {
			t.Errorf("expected podman start, got %v", *calls)
		}
	})

	t.Run("container stopped — starts", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
		})
		_, err := ensureStarted()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "start", "silo-abc12345") {
			t.Errorf("expected podman start, got %v", *calls)
		}
	})

	t.Run("container already running — no action", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		_, err := ensureStarted()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "create") || anyCall(calls, "podman", "start") {
			t.Errorf("expected no create or start, got %v", *calls)
		}
	})
}

func TestEnsureCreatedCreatesContainer(t *testing.T) {
	t.Run("container doesn't exist — creates it", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":     exec.Command("true"),
			"podman image exists silo-abc12345":     exec.Command("true"),
			"podman container exists silo-abc12345": exec.Command("false"),
		})
		_, err := ensureCreated()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "create") {
			t.Errorf("expected podman create, got %v", *calls)
		}
	})
}

func TestEnsureStartedStartsStoppedContainer(t *testing.T) {
	t.Run("container exists but stopped — starts it", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
		})
		_, err := ensureStarted()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "start", "silo-abc12345") {
			t.Errorf("expected podman start, got %v", *calls)
		}
	})
}

func TestStartContainerError(t *testing.T) {
	t.Run("podman start failure", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		// Mock podman start to fail
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman start silo-abc12345": exec.Command("false"),
		})
		err := startContainer("silo-abc12345")
		if err == nil {
			t.Error("expected error when podman start fails")
		}
	})
}

func TestStopContainerError(t *testing.T) {
	t.Run("podman stop failure", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		// Mock podman stop to fail
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman stop -t 0 silo-abc12345": exec.Command("false"),
		})
		err := stopContainer("silo-abc12345")
		if err == nil {
			t.Error("expected error when podman stop fails")
		}
	})
}

// TestEnsureCreatedError is not feasible to test because createContainer calls
// buildContainerArgs which uses os.Getwd() producing dynamic volume mount paths.
// The mock cannot match the full command string with dynamic paths.

func TestEnsureStartedError(t *testing.T) {
	t.Run("startContainer failure returns error", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		// Container exists but stopped, and podman start fails
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
			"podman start silo-abc12345":                                         exec.Command("false"),
		})
		_, err := ensureStarted()
		if err == nil {
			t.Error("expected error when startContainer fails")
		}
	})
}

func TestEnsureStartedWithSharedVolume(t *testing.T) {
	t.Run("ensureStarted succeeds even when container not running initially", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		// Container exists but stopped, podman start succeeds
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
			"podman start silo-abc12345":                                         exec.Command("true"),
		})
		_, err := ensureStarted()
		if err != nil {
			t.Errorf("expected ensureStarted to succeed, got error: %v", err)
		}
	})
}

func TestRemoveContainerError(t *testing.T) {
	t.Run("removeContainer returns error on podman rm failure", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman rm -f silo-abc12345": exec.Command("false"),
		})
		err := removeContainer("silo-abc12345")
		if err == nil {
			t.Error("expected error when removeContainer fails")
		}
	})
}

func TestRemoveImageError(t *testing.T) {
	t.Run("removeImage returns error on podman rmi failure", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman rmi silo-abc12345": exec.Command("false"),
		})
		err := removeImage("silo-abc12345")
		if err == nil {
			t.Error("expected error when removeImage fails")
		}
	})
}
