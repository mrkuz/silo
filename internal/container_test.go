package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestContainerArgs(t *testing.T) {
	tests := []struct {
		name   string
		podman bool
		suffix string
	}{
		{"basic podman", true, ""},
		{"non-nested", false, ""},
		{"with suffix", false, "-dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := WorkspaceConfig{
				General:  WorkspaceGeneralConfig{ID: "abc12345"},
				Features: FeaturesConfig{Podman: tt.podman},
			}
			args := ContainerArgs(cfg, tt.suffix)
			joined := strings.Join(args, " ")
			wantName := "silo-abc12345" + tt.suffix
			wantHost := "silo-abc12345" + tt.suffix
			if !strings.Contains(joined, "--name "+wantName) {
				t.Errorf("expected --name %s in args: %v", wantName, args)
			}
			if !strings.Contains(joined, "--hostname "+wantHost) {
				t.Errorf("expected --hostname %s in args: %v", wantHost, args)
			}
		})
	}
}

func TestWorkspaceMountPath(t *testing.T) {
	cfg := WorkspaceConfig{
		General: WorkspaceGeneralConfig{ID: "abc12345"},
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

func TestCreateContainerArgsMinimal(t *testing.T) {
	cfg := MergedConfig{
		ID:       "abc12345",
		User:     "alice",
		Features: FeaturesConfig{Podman: false},
	}
	args, err := CreateContainerArgs(cfg)
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

func TestCreateContainerArgsSharedVolume(t *testing.T) {
	cfg := MergedConfig{
		ID:       "abc12345",
		User:     "alice",
		Features: FeaturesConfig{Podman: false},
		Persistence: PersistenceConfig{
			SharedPaths: []string{"$HOME/.cache/uv/"},
		},
	}
	args, err := CreateContainerArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--mount type=volume,source=silo,target=/home/alice/.cache/uv,subpath=shared/home/alice/.cache/uv,z") {
		t.Errorf("expected subpath volume mount in args: %v", args)
	}
}

func TestCreateContainerArgsPrivateVolume(t *testing.T) {
	cfg := MergedConfig{
		ID:       "abc12345",
		User:     "alice",
		Features: FeaturesConfig{Podman: false},
		Persistence: PersistenceConfig{
			PrivatePaths: []string{"$HOME/.local/share/"},
		},
	}
	args, err := CreateContainerArgs(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--mount type=volume,source=silo,target=/home/alice/.local/share,subpath=abc12345/home/alice/.local/share,z") {
		t.Errorf("expected private subpath volume mount in args: %v", args)
	}
}

func TestCreateContainerArgsWithLimits(t *testing.T) {
	t.Run("positive limits add flags", func(t *testing.T) {
		cfg := MergedConfig{
			ID:       "abc12345",
			User:     "alice",
			Features: FeaturesConfig{Podman: false},
			Limits:   LimitsConfig{CPUs: 2, Memory: 1024, Processes: 512},
		}
		args, err := CreateContainerArgs(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		joined := strings.Join(args, " ")
		if !strings.Contains(joined, "--cpus=2") {
			t.Errorf("expected --cpus=2 in args: %v", args)
		}
		if !strings.Contains(joined, "--memory=1024m") {
			t.Errorf("expected --memory=1024m in args: %v", args)
		}
		if !strings.Contains(joined, "--pids-limit=512") {
			t.Errorf("expected --pids-limit=512 in args: %v", args)
		}
	})

	t.Run("zero and negative limits add unlimited flags", func(t *testing.T) {
		for _, val := range []int{0, -1} {
			cfg := MergedConfig{
				ID:       "abc12345",
				User:     "alice",
				Features: FeaturesConfig{Podman: false},
				Limits:   LimitsConfig{CPUs: val, Memory: val, Processes: val},
			}
			args, err := CreateContainerArgs(cfg)
			if err != nil {
				t.Fatalf("unexpected error for value %d: %v", val, err)
			}
			joined := strings.Join(args, " ")
			if !strings.Contains(joined, "--cpus=0") {
				t.Errorf("expected --cpus=0 in args: %v", args)
			}
			if !strings.Contains(joined, "--memory=0") {
				t.Errorf("expected --memory=0 in args: %v", args)
			}
			if !strings.Contains(joined, "--pids-limit=-1") {
				t.Errorf("expected --pids-limit=-1 in args: %v", args)
			}
		}
	})

	t.Run("partial limits add positive and unlimited flags", func(t *testing.T) {
		cfg := MergedConfig{
			ID:       "abc12345",
			User:     "alice",
			Features: FeaturesConfig{Podman: false},
			Limits:   LimitsConfig{CPUs: 4, Memory: 0, Processes: 256},
		}
		args, err := CreateContainerArgs(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		joined := strings.Join(args, " ")
		if !strings.Contains(joined, "--cpus=4") {
			t.Errorf("expected --cpus=4 in args: %v", args)
		}
		if !strings.Contains(joined, "--memory=0") {
			t.Errorf("expected --memory=0 in args: %v", args)
		}
		if !strings.Contains(joined, "--pids-limit=256") {
			t.Errorf("expected --pids-limit=256 in args: %v", args)
		}
	})
}

func TestCreateContainerCreateArgs(t *testing.T) {
	cfg := MinimalMergedConfig("abc12345", "testuser")
	cfg.Podman.CreateArgs = []string{"--memory", "512m"}
	mock := NewMock(t)
	mock.MockExec(map[string]*exec.Cmd{})
	_ = CreateContainer(cfg)
	rec := mock.AssertExec("podman", "create", "<...>")
	if rec != nil {
		if !strings.Contains(rec.String(), "--memory") {
			t.Error("expected --memory in create command")
		}
	}
}

func TestCreateContainerCreateArgsNested(t *testing.T) {
	cfg := MinimalMergedConfig("abc12345", "testuser")
	cfg.Podman.CreateArgs = []string{"--security-opt", "label=disable", "--device", "/dev/fuse"}
	mock := NewMock(t)
	mock.MockExec(map[string]*exec.Cmd{})
	_ = CreateContainer(cfg)
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
	cfg := MinimalMergedConfig("abc12345", "testuser")
	cfg.Podman.CreateArgs = []string{"--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges"}
	mock := NewMock(t)
	mock.MockExec(map[string]*exec.Cmd{})
	_ = CreateContainer(cfg)
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
	t.Run("skipped when paths empty", func(t *testing.T) {
		cfg := MinimalMergedConfig("abc12345", "testuser")
		cfg.Persistence.SharedPaths = []string{}
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{})
		if _, err := VolumeSetup(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mock.AssertNoExec("podman", "run", "<...>")
	})

	t.Run("runs workspace image to create directories", func(t *testing.T) {
		cfg := MinimalMergedConfig("abc12345", "testuser")
		cfg.Persistence.SharedPaths = []string{"$HOME/.cache/uv/"}
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman image exists silo-abc12345": exec.Command("true"),
		})
		_, _ = VolumeSetup(cfg)
		mock.AssertExec("podman", "run", "--rm", "-v", "silo:/silo/persistence:z", "silo-abc12345", "sh", "-c", "<...>")
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
		cfg := MinimalWorkspaceConfig("abc12345")
		cfg.Persistence.SharedPaths = []string{"$HOME/.cache/uv/"}
		SetupWorkspaceFiles(t, cfg)
		SetupUserFiles(t)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
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
		cfg := MinimalWorkspaceConfig("abc12345")
		cfg.Persistence.SharedPaths = []string{"$HOME/.cache/uv/"}
		SetupWorkspaceFiles(t, cfg)
		SetupUserFiles(t)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
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
		cfg := MinimalWorkspaceConfig("abc12345")
		SetupWorkspaceFiles(t, cfg)
		SetupUserFiles(t)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
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
		cfg := MinimalWorkspaceConfig("abc12345")
		SetupWorkspaceFiles(t, cfg)
		SetupUserFiles(t)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
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

func TestStartContainerError(t *testing.T) {
	t.Run("podman start failure", func(t *testing.T) {
		cfg := MinimalWorkspaceConfig("abc12345")
		SetupWorkspaceFiles(t, cfg)
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
		cfg := MinimalWorkspaceConfig("abc12345")
		SetupWorkspaceFiles(t, cfg)
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

func TestEnsureStartedWithSharedVolume(t *testing.T) {
	t.Run("ensureStarted succeeds even when container not running initially", func(t *testing.T) {
		cfg := MinimalWorkspaceConfig("abc12345")
		cfg.Persistence.SharedPaths = []string{"$HOME/.cache/uv/"}
		SetupWorkspaceFiles(t, cfg)
		SetupUserFiles(t)
		mock := NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
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
		cfg := MinimalWorkspaceConfig("abc12345")
		SetupWorkspaceFiles(t, cfg)
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
		cfg := MinimalWorkspaceConfig("abc12345")
		SetupWorkspaceFiles(t, cfg)
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

func TestResolveContainerPath(t *testing.T) {
	tests := []struct {
		path   string
		user   string
		expect string
	}{
		{"/absolute/path", "alice", "/absolute/path"},
		{"/absolute/path/", "alice", "/absolute/path"},
		{"$HOME", "alice", "/home/alice"},
		{"$HOME/.cache", "alice", "/home/alice/.cache"},
		{"$HOME/.config/nvim", "bob", "/home/bob/.config/nvim"},
		{"relative/path", "alice", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.user, func(t *testing.T) {
			got := ResolveContainerPath(tt.path, tt.user)
			if got != tt.expect {
				t.Errorf("ResolveContainerPath(%q, %q) = %q, want %q", tt.path, tt.user, got, tt.expect)
			}
		})
	}
}

func TestDefaultCreateArgs(t *testing.T) {
	t.Run("podman=true returns fuse device and label disable", func(t *testing.T) {
		got := DefaultCreateArgs(true)
		joined := strings.Join(got, " ")
		if !strings.Contains(joined, "--security-opt label=disable") {
			t.Errorf("expected --security-opt label=disable, got %v", got)
		}
		if !strings.Contains(joined, "--device /dev/fuse") {
			t.Errorf("expected --device /dev/fuse, got %v", got)
		}
	})

	t.Run("podman=false returns capability restrictions", func(t *testing.T) {
		got := DefaultCreateArgs(false)
		joined := strings.Join(got, " ")
		if !strings.Contains(joined, "--cap-drop=ALL") {
			t.Errorf("expected --cap-drop=ALL, got %v", got)
		}
		if !strings.Contains(joined, "--cap-add=NET_BIND_SERVICE") {
			t.Errorf("expected --cap-add=NET_BIND_SERVICE, got %v", got)
		}
		if !strings.Contains(joined, "--security-opt no-new-privileges") {
			t.Errorf("expected --security-opt no-new-privileges, got %v", got)
		}
	})
}

func TestContainerNameWithSuffix(t *testing.T) {
	tests := []struct {
		base   string
		suffix string
		want   string
	}{
		{"silo-abc", "", "silo-abc"},
		{"silo-abc", "-dev", "silo-abc-dev"},
		{"silo-abc", "-test", "silo-abc-test"},
	}
	for _, tt := range tests {
		got := containerNameWithSuffix(tt.base, tt.suffix)
		if got != tt.want {
			t.Errorf("containerNameWithSuffix(%q, %q) = %q, want %q", tt.base, tt.suffix, got, tt.want)
		}
	}
}
