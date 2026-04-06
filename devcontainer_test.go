package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDeepMergeJSON(t *testing.T) {
	t.Run("scalar override", func(t *testing.T) {
		base := map[string]any{"a": 1.0}
		overlay := map[string]any{"a": 2.0}
		got := deepMergeJSON(base, overlay)
		if got["a"] != 2.0 {
			t.Errorf("expected a=2, got %v", got["a"])
		}
	})

	t.Run("new key from overlay", func(t *testing.T) {
		base := map[string]any{"a": 1.0}
		overlay := map[string]any{"b": 2.0}
		got := deepMergeJSON(base, overlay)
		if got["a"] != 1.0 || got["b"] != 2.0 {
			t.Errorf("unexpected result: %v", got)
		}
	})

	t.Run("base-only key preserved", func(t *testing.T) {
		base := map[string]any{"a": 1.0, "b": 2.0}
		overlay := map[string]any{"b": 3.0}
		got := deepMergeJSON(base, overlay)
		if got["a"] != 1.0 {
			t.Errorf("expected base key a=1 to be preserved, got %v", got["a"])
		}
	})

	t.Run("nested object merge", func(t *testing.T) {
		base := map[string]any{"obj": map[string]any{"x": 1.0}}
		overlay := map[string]any{"obj": map[string]any{"y": 2.0}}
		got := deepMergeJSON(base, overlay)
		obj, ok := got["obj"].(map[string]any)
		if !ok {
			t.Fatal("expected obj to be a map")
		}
		if obj["x"] != 1.0 || obj["y"] != 2.0 {
			t.Errorf("expected merged obj={x:1 y:2}, got %v", obj)
		}
	})

	t.Run("array concatenation", func(t *testing.T) {
		base := map[string]any{"arr": []any{"a", "b"}}
		overlay := map[string]any{"arr": []any{"c"}}
		got := deepMergeJSON(base, overlay)
		arr, ok := got["arr"].([]any)
		if !ok {
			t.Fatal("expected arr to be a slice")
		}
		want := []any{"a", "b", "c"}
		if !reflect.DeepEqual(arr, want) {
			t.Errorf("expected %v, got %v", want, arr)
		}
	})

	t.Run("overlay scalar replaces base object", func(t *testing.T) {
		base := map[string]any{"k": map[string]any{"x": 1.0}}
		overlay := map[string]any{"k": "string"}
		got := deepMergeJSON(base, overlay)
		if got["k"] != "string" {
			t.Errorf("expected overlay scalar to win, got %v", got["k"])
		}
	})

	t.Run("does not mutate base or overlay", func(t *testing.T) {
		base := map[string]any{"a": 1.0}
		overlay := map[string]any{"a": 2.0}
		deepMergeJSON(base, overlay)
		if base["a"] != 1.0 {
			t.Errorf("base was mutated")
		}
	})
}

func TestCmdDevcontainerSharedVolume(t *testing.T) {
	t.Run("includes mounts and postStart when shared volume enabled", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		setupWorkspace(t, cfg)
		setupUserConfig(t)

		if err := cmdDevcontainerGenerate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(".devcontainer.json")
		if err != nil {
			t.Fatalf("read .devcontainer.json: %v", err)
		}

		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("expected valid json, got error: %v\n%s", err, string(data))
		}

		if _, ok := parsed["mounts"]; !ok {
			t.Fatalf("expected mounts in devcontainer output")
		}
		if parsed["postStartCommand"] != "bash /silo/setup.sh" {
			t.Fatalf("expected postStartCommand to run setup script, got %v", parsed["postStartCommand"])
		}
	})

	t.Run("omits mounts and postStart when shared volume disabled", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		cfg.Features.SharedVolume = false
		setupWorkspace(t, cfg)
		setupUserConfig(t)

		if err := cmdDevcontainerGenerate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path := filepath.Join(".devcontainer.json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("expected valid json, got error: %v\n%s", err, string(data))
		}

		if _, ok := parsed["mounts"]; ok {
			t.Fatalf("did not expect mounts in devcontainer output")
		}
		if _, ok := parsed["postStartCommand"]; ok {
			t.Fatalf("did not expect postStartCommand in devcontainer output")
		}
	})
}

func TestCmdDevcontainerStop(t *testing.T) {
	t.Run("container running — stops it", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
		})
		if err := cmdDevcontainerStop(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "stop", "-t", "0", "silo-abc12345-dev") {
			t.Errorf("expected podman stop for dev container, got %v", *calls)
		}
	})

	t.Run("container not running — no-op", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "false"),
		})
		if err := cmdDevcontainerStop(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "stop") {
			t.Errorf("expected no podman stop, got %v", *calls)
		}
	})
}

func TestCmdDevcontainerStatus(t *testing.T) {
	t.Run("running", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
		})
		if err := cmdDevcontainerStatus(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("stopped", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "false"),
		})
		if err := cmdDevcontainerStatus(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestCmdDevcontainerRemove(t *testing.T) {
	t.Run("running container without --force: returns error", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345-dev":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
		})
		err := cmdDevcontainerRemove([]string{})
		if err == nil {
			t.Fatal("expected error when container is running without --force")
		}
		if !strings.Contains(err.Error(), "is running") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("running container with --force: stops and removes", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345-dev":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
		})
		if err := cmdDevcontainerRemove([]string{"--force"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !anyCall(calls, "podman", "stop", "-t", "0", "silo-abc12345-dev") {
			t.Errorf("expected podman stop, got %v", *calls)
		}
		if !anyCall(calls, "podman", "rm", "-f", "silo-abc12345-dev") {
			t.Errorf("expected podman rm -f, got %v", *calls)
		}
	})

	t.Run("stopped container: removes without stop", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345-dev":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "false"),
		})
		if err := cmdDevcontainerRemove([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "stop") {
			t.Errorf("expected no podman stop, got %v", *calls)
		}
		if !anyCall(calls, "podman", "rm", "-f", "silo-abc12345-dev") {
			t.Errorf("expected podman rm -f, got %v", *calls)
		}
	})

	t.Run("container absent: no remove call", func(t *testing.T) {
		cfg := minimalConfig("abc12345")
		setupWorkspace(t, cfg)
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman container exists silo-abc12345-dev": exec.Command("false"),
		})
		if err := cmdDevcontainerRemove([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if anyCall(calls, "podman", "rm") {
			t.Errorf("expected no podman rm, got %v", *calls)
		}
	})
}

