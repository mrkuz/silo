package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestSecurityArgsNested(t *testing.T) {
	args := securityArgs(true)
	if !contains(strings.Join(args, " "), "label=disable") {
		t.Errorf("nested mode should include label=disable, got %v", args)
	}
	if !contains(strings.Join(args, " "), "/dev/fuse") {
		t.Errorf("nested mode should include /dev/fuse, got %v", args)
	}
}

func TestSecurityArgsNonNested(t *testing.T) {
	args := securityArgs(false)
	joined := strings.Join(args, " ")
	if !contains(joined, "--cap-drop=ALL") {
		t.Errorf("non-nested mode should include --cap-drop=ALL, got %v", args)
	}
	if !contains(joined, "no-new-privileges") {
		t.Errorf("non-nested mode should include no-new-privileges, got %v", args)
	}
	if contains(joined, "label=disable") {
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
	if !contains(joined, "--name silo-abc12345") {
		t.Errorf("expected --name silo-abc12345 in args: %v", args)
	}
	if !contains(joined, "--hostname silo-abc12345") {
		t.Errorf("expected --hostname silo-abc12345 in args: %v", args)
	}
	if !contains(joined, "--user alice") {
		t.Errorf("expected --user alice in args: %v", args)
	}
	if contains(joined, "--volume") {
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
	if !contains(joined, sharedVolume+":/shared:Z") {
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
	if !contains(joined, "/workspace/abc12345/") {
		t.Errorf("expected workspace mount path in args: %v", args)
	}
	if !contains(joined, "--workdir") {
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
	if !contains(joined, "label=disable") {
		t.Errorf("expected nested security args, got: %v", args)
	}
}

func TestParseConnectFlags(t *testing.T) {
	tests := []struct {
		args     []string
		wantStop bool
		wantErr  bool
	}{
		{[]string{}, false, false},
		{[]string{"--stop"}, true, false},
		{[]string{"--unknown"}, false, true},
	}
	for _, tt := range tests {
		f, err := parseConnectFlags(tt.args)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseConnectFlags(%v): expected error", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseConnectFlags(%v): unexpected error: %v", tt.args, err)
			continue
		}
		if f.stop != tt.wantStop {
			t.Errorf("parseConnectFlags(%v).stop = %v, want %v", tt.args, f.stop, tt.wantStop)
		}
	}
}

func TestParseConnectFlagsExtra(t *testing.T) {
	f, err := parseConnectFlags([]string{"--", "arg1", "arg2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.extra) != 2 || f.extra[0] != "arg1" || f.extra[1] != "arg2" {
		t.Errorf("expected extra=[arg1 arg2], got %v", f.extra)
	}
}

func TestParseCreateFlags(t *testing.T) {
	tests := []struct {
		args               []string
		wantNested         bool
		wantNoWS           bool
		wantNoSharedVolume bool
		wantErr            bool
	}{
		{[]string{}, false, false, false, false},
		{[]string{"--nested"}, true, false, false, false},
		{[]string{"--no-workspace"}, false, true, false, false},
		{[]string{"--no-shared-volume"}, false, false, true, false},
		{[]string{"--nested", "--no-shared-volume"}, true, false, true, false},
		{[]string{"--unknown"}, false, false, false, true},
	}
	for _, tt := range tests {
		f, err := parseCreateFlags(tt.args)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseCreateFlags(%v): expected error", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseCreateFlags(%v): unexpected error: %v", tt.args, err)
			continue
		}
		if f.nested != tt.wantNested || f.noWorkspace != tt.wantNoWS || f.noSharedVolume != tt.wantNoSharedVolume {
			t.Errorf("parseCreateFlags(%v) flags = %+v, want nested=%v noWS=%v noSharedVolume=%v",
				tt.args, f, tt.wantNested, tt.wantNoWS, tt.wantNoSharedVolume)
		}
	}
}

func TestParseStartFlags(t *testing.T) {
	tests := []struct {
		args      []string
		wantForce bool
		wantErr   bool
	}{
		{[]string{}, false, false},
		{[]string{"--force"}, true, false},
		{[]string{"--unknown"}, false, true},
	}
	for _, tt := range tests {
		f, err := parseStartFlags(tt.args)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseStartFlags(%v): expected error", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseStartFlags(%v): unexpected error: %v", tt.args, err)
			continue
		}
		if f.force != tt.wantForce {
			t.Errorf("parseStartFlags(%v).force = %v, want %v", tt.args, f.force, tt.wantForce)
		}
	}
}

func TestParseRemoveFlags(t *testing.T) {
	tests := []struct {
		args          []string
		wantRemoveImg bool
		wantErr       bool
	}{
		{[]string{}, false, false},
		{[]string{"--image"}, true, false},
		{[]string{"--unknown"}, false, true},
	}
	for _, tt := range tests {
		got, err := parseRemoveFlags(tt.args)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseRemoveFlags(%v): expected error", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseRemoveFlags(%v): unexpected error: %v", tt.args, err)
			continue
		}
		if got != tt.wantRemoveImg {
			t.Errorf("parseRemoveFlags(%v) = %v, want %v", tt.args, got, tt.wantRemoveImg)
		}
	}
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

func TestEnsureContainerRunning(t *testing.T) {
	t.Run("container absent — creates it", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345": exec.Command("false"),
		})
		// createContainer will also call podman create and podman start; all default to true (success).
		_ = ensureContainerRunning(cfg)
		if !anyCall(calls, "podman", "create") {
			t.Errorf("expected podman create call, got %v", *calls)
		}
	})

	t.Run("container stopped — starts it", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
		})
		if err := ensureContainerRunning(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "start", "silo-abc12345") {
			t.Errorf("expected podman start, got %v", *calls)
		}
	})

	t.Run("container already running — no-op", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		if err := ensureContainerRunning(cfg); err != nil {
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

// ---- cmd* tests -----------------------------------------------------------

func TestCmdStop(t *testing.T) {
	t.Run("container running — stops it", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		if err := cmdStop(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "stop", "-t", "0", "silo-abc12345") {
			t.Errorf("expected podman stop, got %v", *calls)
		}
	})

	t.Run("container not running — no-op", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
		})
		if err := cmdStop(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "stop") {
			t.Errorf("expected no podman stop, got %v", *calls)
		}
	})
}

func TestCmdStatus(t *testing.T) {
	t.Run("running", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		if err := cmdStatus(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("stopped", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
		})
		if err := cmdStatus(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestCmdRemove(t *testing.T) {
	t.Run("running container with --image: stops, removes container and image", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
		})
		if err := cmdRemove([]string{"--image"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "stop") {
			t.Errorf("expected podman stop, got %v", *calls)
		}
		if !anyCall(calls, "podman", "rm", "-f", "silo-abc12345") {
			t.Errorf("expected podman rm -f, got %v", *calls)
		}
		if !anyCall(calls, "podman", "rmi", "silo-abc12345") {
			t.Errorf("expected podman rmi, got %v", *calls)
		}
	})

	t.Run("stopped container, no --image: removes container only", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
		})
		if err := cmdRemove([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "stop") {
			t.Errorf("expected no podman stop, got %v", *calls)
		}
		if !anyCall(calls, "podman", "rm", "-f", "silo-abc12345") {
			t.Errorf("expected podman rm -f, got %v", *calls)
		}
		if anyCall(calls, "podman", "rmi") {
			t.Errorf("expected no podman rmi, got %v", *calls)
		}
	})

	t.Run("container absent: no remove call", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345": exec.Command("false"),
		})
		if err := cmdRemove([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "rm") {
			t.Errorf("expected no podman rm, got %v", *calls)
		}
	})
}

func TestCmdExec(t *testing.T) {
	t.Run("container running — runs exec", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		_ = cmdExec([]string{"/bin/sh"})
		if !anyCall(calls, "podman", "exec", "-ti", "silo-abc12345", "/bin/sh") {
			t.Errorf("expected podman exec -ti silo-abc12345 /bin/sh, got %v", *calls)
		}
	})

	t.Run("container not running — returns error", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
		})
		err := cmdExec([]string{"/bin/sh"})
		if err == nil {
			t.Error("expected error when container not running")
		}
		if !strings.Contains(err.Error(), "not running") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestCmdStart(t *testing.T) {
	t.Run("container not running — starts it", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupGlobalConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
		})
		if err := cmdStart([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "start", "silo-abc12345") {
			t.Errorf("expected podman start, got %v", *calls)
		}
	})

	t.Run("already running without --force — no-op", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupGlobalConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		if err := cmdStart([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "start") {
			t.Errorf("expected no podman start, got %v", *calls)
		}
	})

	t.Run("already running with --force — stops then starts", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupGlobalConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
		})
		if err := cmdStart([]string{"--force"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "stop", "-t", "0", "silo-abc12345") {
			t.Errorf("expected podman stop, got %v", *calls)
		}
		if !anyCall(calls, "podman", "start", "silo-abc12345") {
			t.Errorf("expected podman start, got %v", *calls)
		}
	})
}

func TestCmdConnect(t *testing.T) {
	t.Run("basic connect — runs podman exec", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupGlobalConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		_ = cmdConnect([]string{})
		if !anyCall(calls, "podman", "exec", "-ti", "silo-abc12345") {
			t.Errorf("expected podman exec -ti silo-abc12345, got %v", *calls)
		}
	})

	t.Run("with --stop — stops container after session", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupGlobalConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		_ = cmdConnect([]string{"--stop"})
		if !anyCall(calls, "podman", "stop", "-t", "0", "silo-abc12345") {
			t.Errorf("expected podman stop after connect, got %v", *calls)
		}
	})
}

func TestCmdCreateFilePersistence(t *testing.T) {
	t.Run("--nested persists Features.Nested=true to silo.toml", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupGlobalConfig(t)
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":     exec.Command("true"),
			"podman image exists silo-abc12345":     exec.Command("true"),
			"podman container exists silo-abc12345": exec.Command("false"),
		})
		if err := cmdCreate([]string{"--nested"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		saved, err := parseTOML(siloToml)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if !saved.Features.Nested {
			t.Error("expected Features.Nested=true in saved silo.toml")
		}
	})

	t.Run("--no-workspace persists Features.Workspace=false to silo.toml", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.Workspace = true
		setupWorkspace(t, cfg)
		setupGlobalConfig(t)
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":     exec.Command("true"),
			"podman image exists silo-abc12345":     exec.Command("true"),
			"podman container exists silo-abc12345": exec.Command("false"),
		})
		if err := cmdCreate([]string{"--no-workspace"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		saved, err := parseTOML(siloToml)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if saved.Features.Workspace {
			t.Error("expected Features.Workspace=false in saved silo.toml")
		}
	})

	t.Run("-- extra args persisted to silo.toml", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupGlobalConfig(t)
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":     exec.Command("true"),
			"podman image exists silo-abc12345":     exec.Command("true"),
			"podman container exists silo-abc12345": exec.Command("false"),
		})
		if err := cmdCreate([]string{"--", "--memory", "512m"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		saved, err := parseTOML(siloToml)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if len(saved.Create.ExtraArgs) != 2 || saved.Create.ExtraArgs[0] != "--memory" || saved.Create.ExtraArgs[1] != "512m" {
			t.Errorf("expected ExtraArgs [--memory 512m], got %v", saved.Create.ExtraArgs)
		}
	})

	t.Run("--dry-run does not write silo.toml changes", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupGlobalConfig(t)
		// Read original mtime to detect any write.
		info, err := os.Stat(siloToml)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		origMod := info.ModTime()
		mockExecCommand(t, map[string]*exec.Cmd{})
		if err := cmdCreate([]string{"--nested", "--dry-run"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		info, err = os.Stat(siloToml)
		if err != nil {
			t.Fatalf("stat after dry-run: %v", err)
		}
		if !info.ModTime().Equal(origMod) {
			t.Error("silo.toml was modified by --dry-run (expected no write)")
		}
	})
}
