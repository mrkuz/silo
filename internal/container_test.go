package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestContainerArgsBasic(t *testing.T) {
	cfg := Config{
		General:  GeneralConfig{ID: "abc12345", User: "testuser"},
		Features: FeaturesConfig{Podman: true},
	}
	args := ContainerArgs(cfg)
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
}

func TestContainerArgsNonNested(t *testing.T) {
	cfg := Config{
		General:  GeneralConfig{ID: "abc12345", User: "testuser"},
		Features: FeaturesConfig{Podman: false},
	}
	args := ContainerArgs(cfg)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--name silo-abc12345") {
		t.Errorf("expected --name silo-abc12345 in args: %v", args)
	}
	if !strings.Contains(joined, "--hostname silo-abc12345") {
		t.Errorf("expected --hostname silo-abc12345 in args: %v", args)
	}
}

func TestContainerArgsNameSuffix(t *testing.T) {
	cfg := Config{
		General:  GeneralConfig{ID: "abc12345"},
		Features: FeaturesConfig{Podman: false},
	}
	args := ContainerArgs(cfg, "-dev")
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
	got, err := WorkspaceMountPath(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cwd, _ := os.Getwd()
	want := "/workspace/abc12345/" + filepath.Base(cwd)
	if got != want {
		t.Errorf("WorkspaceMountPath() = %q, want %q", got, want)
	}
}

func TestBuildContainerArgsMinimal(t *testing.T) {
	cfg := Config{
		General: GeneralConfig{
			ID:   "abc12345",
			User: "alice",
		},
		Features: FeaturesConfig{
			SharedVolume: false,
			Podman:       false,
		},
	}
	args, err := BuildContainerArgs(cfg)
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
			ID:   "abc12345",
			User: "alice",
		},
		Features: FeaturesConfig{
			SharedVolume: true,
			Podman:       false,
		},
	}
	args, err := BuildContainerArgs(cfg)
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
			ID:   "abc12345",
			User: "alice",
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
	args, err := BuildContainerArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--mount type=volume,source=silo-shared,target=/home/alice/.cache/uv,subpath=home/alice/.cache/uv,Z") {
		t.Errorf("expected subpath volume mount in args: %v", args)
	}
}

func TestCreateContainerCreateArgs(t *testing.T) {
	cfg := MinimalConfig("abc12345")
	cfg.Podman.CreateArgs = []string{"--memory", "512m"}
	mock := NewMock(t)
	mock.MockExec(map[string]*exec.Cmd{})
	_ = CreateContainer(cfg, cfg.Podman.CreateArgs)
	rec := mock.AssertExec("podman", "create", "<...>")
	if rec != nil {
		if !strings.Contains(rec.String(), "--memory") {
			t.Error("expected --memory in create command")
		}
	}
}

func TestCreateContainerCreateArgsNested(t *testing.T) {
	cfg := MinimalConfig("abc12345")
	cfg.Podman.CreateArgs = []string{"--security-opt", "label=disable", "--device", "/dev/fuse"}
	mock := NewMock(t)
	mock.MockExec(map[string]*exec.Cmd{})
	_ = CreateContainer(cfg, cfg.Podman.CreateArgs)
	// Verify the specific arguments - check each appears somewhere in the command
	rec := mock.AssertExec("podman", "create", "<...>")
	if rec != nil {
		cmdStr := rec.String()
		if !strings.Contains(cmdStr, "--security-opt") {
			t.Error("expected --security-opt in create command")
		}
		if !strings.Contains(cmdStr, "--device") {
			t.Error("expected --device in create command")
		}
	}
}

func TestCreateContainerCreateArgsNonNested(t *testing.T) {
	cfg := MinimalConfig("abc12345")
	cfg.Podman.CreateArgs = []string{"--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges"}
	mock := NewMock(t)
	mock.MockExec(map[string]*exec.Cmd{})
	_ = CreateContainer(cfg, cfg.Podman.CreateArgs)
	// Verify the specific arguments - check each appears in the command
	rec := mock.AssertExec("podman", "create", "<...>")
	if rec != nil {
		cmdStr := rec.String()
		if !strings.Contains(cmdStr, "--cap-drop=ALL") {
			t.Error("expected --cap-drop=ALL in create command")
		}
		if !strings.Contains(cmdStr, "--cap-add=NET_BIND_SERVICE") {
			t.Error("expected --cap-add=NET_BIND_SERVICE in create command")
		}
		if !strings.Contains(cmdStr, "--security-opt") {
			t.Error("expected --security-opt in create command")
		}
	}
}

func TestVolumeSetup(t *testing.T) {
	t.Run("skipped when shared volume disabled", func(t *testing.T) {
		cfg := MinimalConfig("abc12345")
		cfg.Features.SharedVolume = false
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{})
		if _, err := VolumeSetup(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mock.AssertNoExec("podman", "run", "<...>")
	})

	t.Run("skipped when paths empty", func(t *testing.T) {
		cfg := MinimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{}
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{})
		if _, err := VolumeSetup(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mock.AssertNoExec("podman", "run", "<...>")
	})

	t.Run("runs user image to create directories", func(t *testing.T) {
		cfg := MinimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{})
		_, _ = VolumeSetup(cfg)
		mock.AssertExec("podman", "run", "--rm", "-v", "silo-shared:/silo/shared:z", "silo-testuser", "sh", "-c", "<...>")
	})
}

func TestContainerExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman container exists silo-abc": exec.Command("true"),
		})
		if !ContainerExists("silo-abc") {
			t.Error("expected ContainerExists to return true")
		}
	})

	t.Run("not exists", func(t *testing.T) {
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman container exists silo-abc": exec.Command("false"),
		})
		if ContainerExists("silo-abc") {
			t.Error("expected ContainerExists to return false")
		}
	})
}

func TestContainerRunning(t *testing.T) {
	t.Run("running", func(t *testing.T) {
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc": exec.Command("echo", "true"),
		})
		if !ContainerRunning("silo-abc") {
			t.Error("expected ContainerRunning to return true")
		}
	})

	t.Run("not running", func(t *testing.T) {
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc": exec.Command("echo", "false"),
		})
		if ContainerRunning("silo-abc") {
			t.Error("expected ContainerRunning to return false")
		}
	})

	t.Run("podman error", func(t *testing.T) {
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc": exec.Command("false"),
		})
		if ContainerRunning("silo-abc") {
			t.Error("expected ContainerRunning false on error")
		}
	})
}

func TestEnsureChain(t *testing.T) {
	t.Run("container absent — creates and starts", func(t *testing.T) {
		cfg := MinimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
		SetupWorkspace(t, cfg)
		SetupUserConfig(t)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists silo-testuser":     exec.Command("true"),
			"podman image exists silo-abc12345":     exec.Command("true"),
			"podman container exists silo-abc12345": exec.Command("false"),
		})
		_, err := EnsureStarted()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mock.AssertExec("podman", "create", "<...>")
		mock.AssertExec("podman", "start", "silo-abc12345")
	})

	t.Run("container stopped — starts", func(t *testing.T) {
		cfg := MinimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
		SetupWorkspace(t, cfg)
		SetupUserConfig(t)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
		})
		_, err := EnsureStarted()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mock.AssertExec("podman", "start", "silo-abc12345")
	})

	t.Run("container already running — no action", func(t *testing.T) {
		cfg := MinimalConfig("abc12345")
		SetupWorkspace(t, cfg)
		SetupUserConfig(t)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		_, err := EnsureStarted()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mock.AssertNoExec("podman", "create", "<...>")
		mock.AssertNoExec("podman", "start", "<...>")
	})
}

func TestEnsureCreatedCreatesContainer(t *testing.T) {
	t.Run("container doesn't exist — creates it", func(t *testing.T) {
		cfg := MinimalConfig("abc12345")
		SetupWorkspace(t, cfg)
		SetupUserConfig(t)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists silo-testuser":     exec.Command("true"),
			"podman image exists silo-abc12345":     exec.Command("true"),
			"podman container exists silo-abc12345": exec.Command("false"),
		})
		_, err := EnsureCreated()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mock.AssertExec("podman", "create", "<...>")
	})
}

func TestEnsureStartedStartsStoppedContainer(t *testing.T) {
	t.Run("container exists but stopped — starts it", func(t *testing.T) {
		cfg := MinimalConfig("abc12345")
		SetupWorkspace(t, cfg)
		SetupUserConfig(t)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
		})
		_, err := EnsureStarted()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mock.AssertExec("podman", "start", "silo-abc12345")
	})
}

func TestStartContainerError(t *testing.T) {
	t.Run("podman start failure", func(t *testing.T) {
		cfg := MinimalConfig("abc12345")
		SetupWorkspace(t, cfg)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman start silo-abc12345": exec.Command("false"),
		})
		err := StartContainer("silo-abc12345")
		if err == nil {
			t.Error("expected error when podman start fails")
		}
	})
}

func TestStopContainerError(t *testing.T) {
	t.Run("podman stop failure", func(t *testing.T) {
		cfg := MinimalConfig("abc12345")
		SetupWorkspace(t, cfg)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman stop -t 0 silo-abc12345": exec.Command("false"),
		})
		err := StopContainer("silo-abc12345")
		if err == nil {
			t.Error("expected error when podman stop fails")
		}
	})
}

func TestEnsureStartedError(t *testing.T) {
	t.Run("startContainer failure returns error", func(t *testing.T) {
		cfg := MinimalConfig("abc12345")
		SetupWorkspace(t, cfg)
		SetupUserConfig(t)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
			"podman start silo-abc12345":                                         exec.Command("false"),
		})
		_, err := EnsureStarted()
		if err == nil {
			t.Error("expected error when startContainer fails")
		}
	})
}

func TestEnsureStartedWithSharedVolume(t *testing.T) {
	t.Run("ensureStarted succeeds even when container not running initially", func(t *testing.T) {
		cfg := MinimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
		SetupWorkspace(t, cfg)
		SetupUserConfig(t)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
			"podman start silo-abc12345":                                         exec.Command("true"),
		})
		_, err := EnsureStarted()
		if err != nil {
			t.Errorf("expected EnsureStarted to succeed, got error: %v", err)
		}
	})
}

func TestRemoveContainerError(t *testing.T) {
	t.Run("removeContainer returns error on podman rm failure", func(t *testing.T) {
		cfg := MinimalConfig("abc12345")
		SetupWorkspace(t, cfg)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman rm -f silo-abc12345": exec.Command("false"),
		})
		err := RemoveContainer("silo-abc12345")
		if err == nil {
			t.Error("expected error when removeContainer fails")
		}
	})
}

func TestRemoveImageError(t *testing.T) {
	t.Run("removeImage returns error on podman rmi failure", func(t *testing.T) {
		cfg := MinimalConfig("abc12345")
		SetupWorkspace(t, cfg)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman rmi silo-abc12345": exec.Command("false"),
		})
		err := RemoveImage("silo-abc12345")
		if err == nil {
			t.Error("expected error when removeImage fails")
		}
	})
}
