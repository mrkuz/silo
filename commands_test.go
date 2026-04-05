package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

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
		wantForce          bool
		wantErr            bool
	}{
		{[]string{}, false, false, false, false, false},
		{[]string{"--nested"}, true, false, false, false, false},
		{[]string{"--no-workspace"}, false, true, false, false, false},
		{[]string{"--no-shared-volume"}, false, false, true, false, false},
		{[]string{"--nested", "--no-shared-volume"}, true, false, true, false, false},
		{[]string{"--force"}, false, false, false, true, false},
		{[]string{"-f"}, false, false, false, true, false},
		{[]string{"--unknown"}, false, false, false, false, true},
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
		if f.nested != tt.wantNested || f.noWorkspace != tt.wantNoWS || f.noSharedVolume != tt.wantNoSharedVolume || f.force != tt.wantForce {
			t.Errorf("parseCreateFlags(%v) flags = %+v, want nested=%v noWS=%v noSharedVolume=%v force=%v",
				tt.args, f, tt.wantNested, tt.wantNoWS, tt.wantNoSharedVolume, tt.wantForce)
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
		{[]string{"-f"}, true, false},
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
		args      []string
		wantForce bool
		wantImage bool
		wantErr   bool
	}{
		{[]string{}, false, false, false},
		{[]string{"--image"}, false, true, false},
		{[]string{"--force"}, true, false, false},
		{[]string{"-f"}, true, false, false},
		{[]string{"--force", "--image"}, true, true, false},
		{[]string{"--unknown"}, false, false, true},
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
		if got.force != tt.wantForce || got.image != tt.wantImage {
			t.Errorf("parseRemoveFlags(%v) = %+v, want force=%v image=%v", tt.args, got, tt.wantForce, tt.wantImage)
		}
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
	t.Run("running container without --force: returns error", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		err := cmdRemove([]string{})
		if err == nil {
			t.Fatal("expected error when container is running without --force")
		}
		if !strings.Contains(err.Error(), "is running") {
			t.Errorf("expected error to mention container is running, got: %v", err)
		}
	})

	t.Run("running container with --force --image: stops, removes container and image", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
		})
		if err := cmdRemove([]string{"--force", "--image"}); err != nil {
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

	t.Run("running container with -f: stops and removes", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		if err := cmdRemove([]string{"-f"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "stop") {
			t.Errorf("expected podman stop, got %v", *calls)
		}
		if !anyCall(calls, "podman", "rm", "-f", "silo-abc12345") {
			t.Errorf("expected podman rm -f, got %v", *calls)
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
		setupUserConfig(t)
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
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		if err := cmdStart([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "start") {
			t.Errorf("expected no podman start, got %v", *calls)
		}
	})

	t.Run("already running with --force — stops container", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		if err := cmdStart([]string{"--force"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "stop", "-t", "0", "silo-abc12345") {
			t.Errorf("expected podman stop, got %v", *calls)
		}
	})
}

func TestCmdConnect(t *testing.T) {
	t.Run("basic connect — runs podman exec", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
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
		setupUserConfig(t)
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
		setupUserConfig(t)
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
		setupUserConfig(t)
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
		setupUserConfig(t)
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
		setupUserConfig(t)
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

func TestCmdInit(t *testing.T) {
	t.Run("creates workspace and starter files", func(t *testing.T) {
		setupUserConfig(t)
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)

		mockExecCommand(t, map[string]*exec.Cmd{})
		if err := cmdInit(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(siloToml); os.IsNotExist(err) {
			t.Error("expected .silo/silo.toml to be created")
		}
		if _, err := os.Stat(".silo/home.nix"); os.IsNotExist(err) {
			t.Error("expected .silo/home.nix to be created")
		}
	})

	t.Run("idempotent — does not overwrite existing config", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)

		mockExecCommand(t, map[string]*exec.Cmd{})
		if err := cmdInit(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		saved, err := parseTOML(siloToml)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if saved.General.ID != "abc12345" {
			t.Errorf("expected ID abc12345, got %q", saved.General.ID)
		}
	})
}

func TestCmdSetup(t *testing.T) {
	t.Run("container running — runs setup", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		cfg.SharedVolume.Paths = []string{"$HOME/.cache/uv/"}
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		if err := cmdSetup(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "exec", "silo-abc12345", "bash", "/silo/setup.sh") {
			t.Errorf("expected podman exec for setup, got %v", *calls)
		}
	})

	t.Run("container not running — returns error", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "false"),
		})
		err := cmdSetup()
		if err == nil {
			t.Error("expected error when container not running")
		}
		if !strings.Contains(err.Error(), "not running") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
