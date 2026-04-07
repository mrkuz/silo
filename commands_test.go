package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestParseRunFlags(t *testing.T) {
	tests := []struct {
		args     []string
		wantStop bool
		wantRm   bool
		wantRmi  bool
		wantErr  bool
	}{
		{[]string{}, false, false, false, false},
		{[]string{"--stop"}, true, false, false, false},
		{[]string{"--rm"}, true, true, false, false},
		{[]string{"--rmi"}, true, true, true, false},
		{[]string{"--unknown"}, false, false, false, true},
	}
	for _, tt := range tests {
		f, err := parseRunFlags(tt.args)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseRunFlags(%v): expected error", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseRunFlags(%v): unexpected error: %v", tt.args, err)
			continue
		}
		if f.stop != tt.wantStop || f.rm != tt.wantRm || f.rmi != tt.wantRmi {
			t.Errorf("parseRunFlags(%v) = {stop:%v rm:%v rmi:%v}, want {stop:%v rm:%v rmi:%v}",
				tt.args, f.stop, f.rm, f.rmi, tt.wantStop, tt.wantRm, tt.wantRmi)
		}
	}
}

func TestParseRunFlagsExtra(t *testing.T) {
	f, err := parseRunFlags([]string{"--", "arg1", "arg2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.extra) != 2 || f.extra[0] != "arg1" || f.extra[1] != "arg2" {
		t.Errorf("expected extra=[arg1 arg2], got %v", f.extra)
	}
}

func TestParseCreateFlags(t *testing.T) {
	tests := []struct {
		args             []string
		wantNested       bool
		wantSharedVolume bool
		wantErr          bool
	}{
		{[]string{}, false, false, false},
		{[]string{"--nested"}, true, false, false},
		{[]string{"--shared-volume"}, false, true, false},
		{[]string{"--nested", "--shared-volume"}, true, true, false},
		{[]string{"--unknown"}, false, false, true},
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
		if f.nested != tt.wantNested || f.sharedVolume != tt.wantSharedVolume {
			t.Errorf("parseCreateFlags(%v) flags = %+v, want nested=%v sharedVolume=%v",
				tt.args, f, tt.wantNested, tt.wantSharedVolume)
		}
	}
}

func TestParseRemoveFlags(t *testing.T) {
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
		if got.force != tt.wantForce {
			t.Errorf("parseRemoveFlags(%v).force = %v, want %v", tt.args, got.force, tt.wantForce)
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

	t.Run("running container with --force: stops and removes", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		if err := cmdRemove([]string{"--force"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "stop") {
			t.Errorf("expected podman stop, got %v", *calls)
		}
		if !anyCall(calls, "podman", "rm", "-f", "silo-abc12345") {
			t.Errorf("expected podman rm -f, got %v", *calls)
		}
	})

	t.Run("stopped container: removes container", func(t *testing.T) {
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

func TestParseRemoveImageFlags(t *testing.T) {
	tests := []struct {
		args      []string
		wantForce bool
		wantUser  bool
		wantErr   bool
	}{
		{[]string{}, false, false, false},
		{[]string{"--force"}, true, false, false},
		{[]string{"-f"}, true, false, false},
		{[]string{"--user"}, false, true, false},
		{[]string{"--force", "--user"}, true, true, false},
		{[]string{"-f", "--user"}, true, true, false},
		{[]string{"--unknown"}, false, false, true},
	}
	for _, tt := range tests {
		got, err := parseRemoveImageFlags(tt.args)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseRemoveImageFlags(%v): expected error", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseRemoveImageFlags(%v): unexpected error: %v", tt.args, err)
			continue
		}
		if got.force != tt.wantForce || got.user != tt.wantUser {
			t.Errorf("parseRemoveImageFlags(%v) = {force:%v user:%v}, want {force:%v user:%v}",
				tt.args, got.force, got.user, tt.wantForce, tt.wantUser)
		}
	}
}

func TestCmdRemoveImage(t *testing.T) {
	t.Run("removes workspace image", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-abc12345": exec.Command("true"),
		})
		if err := cmdRemoveImage([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "rmi", "silo-abc12345") {
			t.Errorf("expected podman rmi, got %v", *calls)
		}
	})

	t.Run("image not found — no-op", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-abc12345": exec.Command("false"),
		})
		if err := cmdRemoveImage([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "rmi") {
			t.Errorf("expected no podman rmi, got %v", *calls)
		}
	})

	t.Run("--force stops and removes container first", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
		})
		if err := cmdRemoveImage([]string{"--force"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "stop", "-t", "0", "silo-abc12345") {
			t.Errorf("expected podman stop, got %v", *calls)
		}
		if !anyCall(calls, "podman", "rm", "-f", "silo-abc12345") {
			t.Errorf("expected podman rm -f, got %v", *calls)
		}
		if !anyCall(calls, "podman", "rmi", "silo-abc12345") {
			t.Errorf("expected podman rmi, got %v", *calls)
		}
	})

	t.Run("--user removes only user image", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-abc12345": exec.Command("true"),
			"podman image exists silo-testuser": exec.Command("true"),
		})
		if err := cmdRemoveImage([]string{"--user"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "rmi", "silo-abc12345") {
			t.Errorf("expected no workspace image remove, got %v", *calls)
		}
		if !anyCall(calls, "podman", "rmi", "silo-testuser") {
			t.Errorf("expected podman rmi for user image, got %v", *calls)
		}
		if anyCall(calls, "podman", "stop") || anyCall(calls, "podman", "rm", "-f", "silo-abc12345") {
			t.Errorf("expected no container stop/remove, got %v", *calls)
		}
	})

	t.Run("--force with --user still removes only user image", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser": exec.Command("true"),
		})
		if err := cmdRemoveImage([]string{"--force", "--user"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "rmi", "silo-testuser") {
			t.Errorf("expected podman rmi for user image, got %v", *calls)
		}
		if anyCall(calls, "podman", "rmi", "silo-abc12345") {
			t.Errorf("expected no workspace image remove, got %v", *calls)
		}
		if anyCall(calls, "podman", "stop") || anyCall(calls, "podman", "rm", "-f", "silo-abc12345") {
			t.Errorf("expected no container stop/remove, got %v", *calls)
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
		if err := cmdStart(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "start", "silo-abc12345") {
			t.Errorf("expected podman start, got %v", *calls)
		}
	})

	t.Run("already running — does not restart", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		if err := cmdStart(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "stop", "-t", "0", "silo-abc12345") {
			t.Errorf("expected no podman stop, got %v", *calls)
		}
		if anyCall(calls, "podman", "start", "silo-abc12345") {
			t.Errorf("expected no podman start, got %v", *calls)
		}
		if anyCall(calls, "podman", "exec", "silo-abc12345", "bash", "/silo/setup.sh") {
			t.Errorf("expected no setup exec call, got %v", *calls)
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

	t.Run("with arguments — returns error", func(t *testing.T) {
		if err := cmdConnect([]string{"--stop"}); err == nil {
			t.Fatal("expected error when passing arguments to connect")
		}
	})
}

func TestCmdRun(t *testing.T) {
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
		_ = cmdRun([]string{"--stop"})
		if !anyCall(calls, "podman", "stop", "-t", "0", "silo-abc12345") {
			t.Errorf("expected podman stop after run, got %v", *calls)
		}
	})

	t.Run("with --rm — stops and removes container after session", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		_ = cmdRun([]string{"--rm"})
		if !anyCall(calls, "podman", "stop", "-t", "0", "silo-abc12345") {
			t.Errorf("expected podman stop, got %v", *calls)
		}
		if !anyCall(calls, "podman", "rm", "-f", "silo-abc12345") {
			t.Errorf("expected podman rm -f, got %v", *calls)
		}
		if anyCall(calls, "podman", "rmi") {
			t.Errorf("expected no podman rmi, got %v", *calls)
		}
	})

	t.Run("with --rmi — stops, removes container, and removes image after session", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":                                  exec.Command("true"),
			"podman image exists silo-abc12345":                                  exec.Command("true"),
			"podman container exists silo-abc12345":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345": exec.Command("echo", "true"),
		})
		_ = cmdRun([]string{"--rmi"})
		if !anyCall(calls, "podman", "stop", "-t", "0", "silo-abc12345") {
			t.Errorf("expected podman stop, got %v", *calls)
		}
		if !anyCall(calls, "podman", "rm", "-f", "silo-abc12345") {
			t.Errorf("expected podman rm -f, got %v", *calls)
		}
		if !anyCall(calls, "podman", "rmi", "silo-abc12345") {
			t.Errorf("expected podman rmi, got %v", *calls)
		}
	})
}

func TestCmdCreateFilePersistence(t *testing.T) {
	t.Run("existing container: no remove and no create", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser":     exec.Command("true"),
			"podman image exists silo-abc12345":     exec.Command("true"),
			"podman container exists silo-abc12345": exec.Command("true"),
		})
		if err := cmdCreate([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "rm", "-f", "silo-abc12345") {
			t.Errorf("expected no podman rm -f for existing container, got %v", *calls)
		}
		if anyCall(calls, "podman", "create") {
			t.Errorf("expected no podman create for existing container, got %v", *calls)
		}
	})

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
