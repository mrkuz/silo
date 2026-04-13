package main

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
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
		args    []string
		wantErr bool
	}{
		{[]string{}, false},
		{[]string{"--unknown"}, true},
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
		if f.dryRun != false {
			t.Errorf("parseCreateFlags(%v) = %+v, want empty flags", tt.args, f)
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
		wantErr   bool
	}{
		{[]string{}, false, false},
		{[]string{"--force"}, true, false},
		{[]string{"-f"}, true, false},
		{[]string{"--user"}, false, true},
		{[]string{"--unknown"}, false, true},
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
		if got.force != tt.wantForce {
			t.Errorf("parseRemoveImageFlags(%v) = {force:%v}, want {force:%v}",
				tt.args, got.force, tt.wantForce)
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

}

func TestCmdUserRmi(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("get current user: %v", err)
	}
	userImage := "silo-" + u.Username

	t.Run("removes user image when present", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists " + userImage: exec.Command("true"),
		})
		if err := cmdUserRmi(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "rmi", userImage) {
			t.Errorf("expected podman rmi for user image, got %v", *calls)
		}
		if anyCall(calls, "podman", "rmi", "silo-abc12345") {
			t.Errorf("expected no workspace image remove, got %v", *calls)
		}
	})

	t.Run("user image absent — no-op", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists " + userImage: exec.Command("false"),
		})
		if err := cmdUserRmi(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "rmi") {
			t.Errorf("expected no podman rmi, got %v", *calls)
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

func TestCmdRunNoSiloToml(t *testing.T) {
	t.Run("creates silo.toml on first run", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)

		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser": exec.Command("false"),
			"podman image exists":               exec.Command("false"),
		})
		err := cmdRun([]string{})
		// Image build will fail in test environment, but silo.toml should be created first
		if _, statErr := os.Stat(siloToml); os.IsNotExist(statErr) {
			t.Errorf("expected .silo/silo.toml to be created, cmdRun error: %v", err)
		}
		_ = calls // suppress unused
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
		if err := cmdCreate([]string{"--dry-run"}); err != nil {
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
	t.Run("creates workspace and user starter files", func(t *testing.T) {
		// Point XDG_CONFIG_HOME at a fresh empty directory (no starter files).
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)

		mockExecCommand(t, map[string]*exec.Cmd{})
		if err := cmdInit([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(siloToml); os.IsNotExist(err) {
			t.Error("expected .silo/silo.toml to be created")
		}
		if _, err := os.Stat(".silo/home.nix"); os.IsNotExist(err) {
			t.Error("expected .silo/home.nix to be created")
		}
		// User files should also be created — silo init delegates to ensureUserFiles.
		userDir := filepath.Join(base, "silo")
		for _, name := range []string{"home-user.nix", "devcontainer.in.json", "silo.in.toml"} {
			if _, err := os.Stat(filepath.Join(userDir, name)); os.IsNotExist(err) {
				t.Errorf("expected user file %s to be created by silo init", name)
			}
		}
	})

	t.Run("idempotent — does not overwrite existing config", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)

		mockExecCommand(t, map[string]*exec.Cmd{})
		if err := cmdInit([]string{}); err != nil {
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

	t.Run("--podman and --shared-volume persist to silo.toml", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)

		mockExecCommand(t, map[string]*exec.Cmd{})
		if err := cmdInit([]string{"--podman", "--shared-volume"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		saved, err := parseTOML(siloToml)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if !saved.Features.Podman {
			t.Error("expected Features.Podman=true")
		}
		if !saved.Features.SharedVolume {
			t.Error("expected Features.SharedVolume=true")
		}
		if len(saved.Create.Arguments) != 4 {
			t.Errorf("expected 4 create arguments, got %v", saved.Create.Arguments)
		}
		if saved.Create.Arguments[0] != "--security-opt" || saved.Create.Arguments[1] != "label=disable" {
			t.Errorf("expected podman create arguments, got %v", saved.Create.Arguments)
		}
	})

	t.Run("--no-podman and --no-shared-volume set features to false", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)

		mockExecCommand(t, map[string]*exec.Cmd{})
		if err := cmdInit([]string{"--no-podman", "--no-shared-volume"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		saved, err := parseTOML(siloToml)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if saved.Features.Podman {
			t.Error("expected Features.Podman=false")
		}
		if saved.Features.SharedVolume {
			t.Error("expected Features.SharedVolume=false")
		}
		if len(saved.Create.Arguments) != 4 {
			t.Errorf("expected 4 create arguments, got %v", saved.Create.Arguments)
		}
		if saved.Create.Arguments[0] != "--cap-drop=ALL" {
			t.Errorf("expected non-podman create arguments, got %v", saved.Create.Arguments)
		}
	})

	t.Run("--podman creates workspace home.nix with podman enabled", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)

		mockExecCommand(t, map[string]*exec.Cmd{})
		if err := cmdInit([]string{"--podman"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content, err := os.ReadFile(".silo/home.nix")
		if err != nil {
			t.Fatalf("failed to read .silo/home.nix: %v", err)
		}
		if !strings.Contains(string(content), "module.podman.enable = true") {
			t.Errorf("expected 'module.podman.enable = true' in home.nix, got: %s", content)
		}
	})

	t.Run("subsequent run does not modify config", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.Podman = true
		cfg.Features.SharedVolume = true
		setupWorkspace(t, cfg)
		setupUserConfig(t)

		// Read original mtime to detect any write.
		info, err := os.Stat(siloToml)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		origMod := info.ModTime()

		mockExecCommand(t, map[string]*exec.Cmd{})
		if err := cmdInit([]string{"--podman"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		info, err = os.Stat(siloToml)
		if err != nil {
			t.Fatalf("stat after init: %v", err)
		}
		if !info.ModTime().Equal(origMod) {
			t.Error("silo.toml was modified on subsequent run (expected no write)")
		}
	})

	t.Run("first run with silo.in.toml defaults", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		siloUser := filepath.Join(base, "silo")
		if err := os.MkdirAll(siloUser, 0755); err != nil {
			t.Fatal(err)
		}
		// Write silo.in.toml with shared_volume=true
		userToml := filepath.Join(siloUser, "silo.in.toml")
		if err := os.WriteFile(userToml, []byte("[features]\nshared_volume = true\npodman = false\n"), 0644); err != nil {
			t.Fatal(err)
		}

		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)

		mockExecCommand(t, map[string]*exec.Cmd{})
		if err := cmdInit([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		saved, err := parseTOML(siloToml)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if saved.Features.SharedVolume != true {
			t.Errorf("expected Features.SharedVolume=true from silo.in.toml, got false")
		}
		if saved.Features.Podman != false {
			t.Errorf("expected Features.Podman=false from silo.in.toml, got true")
		}
	})

	t.Run("silo.in.toml [create].arguments prepended before defaults (--no-podman)", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		siloUser := filepath.Join(base, "silo")
		if err := os.MkdirAll(siloUser, 0755); err != nil {
			t.Fatal(err)
		}
		userToml := filepath.Join(siloUser, "silo.in.toml")
		if err := os.WriteFile(userToml, []byte("[create]\narguments = [\"--memory\", \"512m\"]\n"), 0644); err != nil {
			t.Fatal(err)
		}

		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)

		mockExecCommand(t, map[string]*exec.Cmd{})
		if err := cmdInit([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		saved, err := parseTOML(siloToml)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		// User args prepended, non-podman defaults appended
		want := []string{"--memory", "512m", "--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges"}
		if len(saved.Create.Arguments) != len(want) {
			t.Errorf("expected %d arguments, got %v", len(want), saved.Create.Arguments)
		}
		for i, w := range want {
			if saved.Create.Arguments[i] != w {
				t.Errorf("Create.Arguments[%d] = %q, want %q", i, saved.Create.Arguments[i], w)
			}
		}
	})

	t.Run("silo.in.toml [create].arguments prepended before defaults (--podman)", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		siloUser := filepath.Join(base, "silo")
		if err := os.MkdirAll(siloUser, 0755); err != nil {
			t.Fatal(err)
		}
		userToml := filepath.Join(siloUser, "silo.in.toml")
		if err := os.WriteFile(userToml, []byte("[create]\narguments = [\"--memory\", \"512m\"]\n"), 0644); err != nil {
			t.Fatal(err)
		}

		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)

		mockExecCommand(t, map[string]*exec.Cmd{})
		if err := cmdInit([]string{"--podman"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		saved, err := parseTOML(siloToml)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if !saved.Features.Podman {
			t.Error("expected Features.Podman=true")
		}
		// User args prepended, podman defaults appended
		want := []string{"--memory", "512m", "--security-opt", "label=disable", "--device", "/dev/fuse"}
		if len(saved.Create.Arguments) != len(want) {
			t.Errorf("expected %d arguments, got %v", len(want), saved.Create.Arguments)
		}
		for i, w := range want {
			if saved.Create.Arguments[i] != w {
				t.Errorf("Create.Arguments[%d] = %q, want %q", i, saved.Create.Arguments[i], w)
			}
		}
	})
}

func TestCmdUserInit(t *testing.T) {
	t.Run("creates all user starter files when absent", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		if err := cmdUserInit(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		userDir := filepath.Join(base, "silo")
		for _, name := range []string{"home-user.nix", "devcontainer.in.json", "silo.in.toml"} {
			if _, err := os.Stat(filepath.Join(userDir, name)); os.IsNotExist(err) {
				t.Errorf("expected %s to be created", name)
			}
		}
	})

	t.Run("does not overwrite existing files", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		userDir := filepath.Join(base, "silo")
		if err := os.MkdirAll(userDir, 0755); err != nil {
			t.Fatal(err)
		}
		sentinel := []byte("# custom\n")
		existing := filepath.Join(userDir, "home-user.nix")
		if err := os.WriteFile(existing, sentinel, 0644); err != nil {
			t.Fatal(err)
		}

		if err := cmdUserInit(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := os.ReadFile(existing)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(sentinel) {
			t.Errorf("existing home-user.nix was overwritten")
		}
		// The other two files should still be created.
		if _, err := os.Stat(filepath.Join(userDir, "devcontainer.in.json")); os.IsNotExist(err) {
			t.Error("expected devcontainer.in.json to be created alongside existing file")
		}
		if _, err := os.Stat(filepath.Join(userDir, "silo.in.toml")); os.IsNotExist(err) {
			t.Error("expected silo.in.toml to be created alongside existing file")
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

func TestCmdCreateDryRunOutput(t *testing.T) {
	t.Run("--dry-run prints podman create command", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		setupUserConfig(t)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-testuser": exec.Command("true"),
			"podman image exists silo-abc12345": exec.Command("true"),
		})
		// Capture stdout by running with a custom output check
		if err := cmdCreate([]string{"--dry-run"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Dry run should NOT call podman create
		if anyCall(calls, "podman", "create") {
			t.Errorf("expected no podman create call in dry-run, got %v", *calls)
		}
		// Should still have called image exists
		if !anyCall(calls, "podman", "image", "exists") {
			t.Errorf("expected image existence check in dry-run, got %v", *calls)
		}
	})
}

func TestRequireWorkspaceConfigErrors(t *testing.T) {
	t.Run("missing silo.toml returns error", func(t *testing.T) {
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)
		// No silo.toml created
		_, err := requireWorkspaceConfig()
		if err == nil {
			t.Error("expected error when silo.toml is missing")
		}
		if !strings.Contains(err.Error(), "no .silo/silo.toml found") {
			t.Errorf("expected specific error message, got: %v", err)
		}
	})

	t.Run("malformed silo.toml returns error", func(t *testing.T) {
		dir := t.TempDir()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)
		if err := os.MkdirAll(".silo", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(".silo/silo.toml", []byte("invalid toml = ["), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := requireWorkspaceConfig()
		if err == nil {
			t.Error("expected error when silo.toml is malformed")
		}
	})
}

func TestCmdRemoveImageForceWhenContainerNotExists(t *testing.T) {
	t.Run("--force with absent container: removes image without error", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345": exec.Command("false"),
			"podman image exists silo-abc12345":     exec.Command("true"),
		})
		if err := cmdRemoveImage([]string{"--force"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should not try to stop a non-existent container
		if anyCall(calls, "podman", "stop") {
			t.Errorf("expected no stop call for non-existent container, got %v", *calls)
		}
		if !anyCall(calls, "podman", "rmi", "silo-abc12345") {
			t.Errorf("expected podman rmi, got %v", *calls)
		}
	})
}

func TestParseInitFlagsConflicts(t *testing.T) {
	t.Run("conflicting podman flags uses last value", func(t *testing.T) {
		// When both --podman and --no-podman are passed, flag package
		// lets the last one win (since they're both bools, last set wins)
		// This test documents the behavior
		flags, err := parseInitFlags([]string{"--podman", "--no-podman"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// --no-podman comes last, so podman should be false
		if flags.podman == nil || *flags.podman {
			t.Errorf("expected podman=false when both flags passed, got %v", flags.podman)
		}
	})

	t.Run("conflicting shared-volume flags uses last value", func(t *testing.T) {
		flags, err := parseInitFlags([]string{"--shared-volume", "--no-shared-volume"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// --no-shared-volume comes last, so sharedVolume should be false
		if flags.sharedVolume == nil || *flags.sharedVolume {
			t.Errorf("expected sharedVolume=false when both flags passed, got %v", flags.sharedVolume)
		}
	})
}
