package cmd_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mrkuz/silo/cmd"
	"github.com/mrkuz/silo/internal"
)

func TestDeepMergeJSON(t *testing.T) {
	t.Run("scalar override", func(t *testing.T) {
		base := map[string]any{"a": 1.0}
		overlay := map[string]any{"a": 2.0}
		got := internal.DeepMergeJSON(base, overlay)
		if got["a"] != 2.0 {
			t.Errorf("expected a=2, got %v", got["a"])
		}
	})

	t.Run("new key from overlay", func(t *testing.T) {
		base := map[string]any{"a": 1.0}
		overlay := map[string]any{"b": 2.0}
		got := internal.DeepMergeJSON(base, overlay)
		if got["a"] != 1.0 || got["b"] != 2.0 {
			t.Errorf("unexpected result: %v", got)
		}
	})

	t.Run("base-only key preserved", func(t *testing.T) {
		base := map[string]any{"a": 1.0, "b": 2.0}
		overlay := map[string]any{"b": 3.0}
		got := internal.DeepMergeJSON(base, overlay)
		if got["a"] != 1.0 {
			t.Errorf("expected base key a=1 to be preserved, got %v", got["a"])
		}
	})

	t.Run("nested object merge", func(t *testing.T) {
		base := map[string]any{"obj": map[string]any{"x": 1.0}}
		overlay := map[string]any{"obj": map[string]any{"y": 2.0}}
		got := internal.DeepMergeJSON(base, overlay)
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
		got := internal.DeepMergeJSON(base, overlay)
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
		got := internal.DeepMergeJSON(base, overlay)
		if got["k"] != "string" {
			t.Errorf("expected overlay scalar to win, got %v", got["k"])
		}
	})

	t.Run("does not mutate base or overlay", func(t *testing.T) {
		base := map[string]any{"a": 1.0}
		overlay := map[string]any{"a": 2.0}
		internal.DeepMergeJSON(base, overlay)
		if base["a"] != 1.0 {
			t.Errorf("base was mutated")
		}
	})
}

func TestCmdDevcontainerSharedVolume(t *testing.T) {
	t.Run("includes mounts and postStart when shared volume enabled", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		cfg.Features.SharedVolume = true
		internal.SetupWorkspace(t, cfg)
		internal.SetupUserConfig(t)

		if err := cmd.DevcontainerGenerate(); err != nil {
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
	})

	t.Run("omits mounts when shared volume disabled", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		cfg.Features.SharedVolume = false
		internal.SetupWorkspace(t, cfg)
		internal.SetupUserConfig(t)

		if err := cmd.DevcontainerGenerate(); err != nil {
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
	})
}

func TestCmdDevcontainerStop(t *testing.T) {
	t.Run("container running — stops it", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
		})
		if err := cmd.DevcontainerStop(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mock.AssertExec("podman", "stop", "-t", "0", "silo-abc12345-dev")
	})

	t.Run("container not running — no-op", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "false"),
		})
		if err := cmd.DevcontainerStop(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mock.AssertNoExec("podman", "stop", "<...>")
	})
}

func TestCmdDevcontainerStatus(t *testing.T) {
	t.Run("running", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
		})
		if err := cmd.DevcontainerStatus(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("stopped", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "false"),
		})
		if err := cmd.DevcontainerStatus(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestCmdDevcontainerRemove(t *testing.T) {
	t.Run("running container without --force: returns error", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman container exists silo-abc12345-dev":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
		})
		err := cmd.DevcontainerRemove([]string{})
		if err == nil {
			t.Fatal("expected error when container is running without --force")
		}
		if !strings.Contains(err.Error(), "is running") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("running container with --force: stops and removes", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman container exists silo-abc12345-dev":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "true"),
		})
		if err := cmd.DevcontainerRemove([]string{"--force"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mock.AssertExec("podman", "stop", "-t", "0", "silo-abc12345-dev")
		mock.AssertExec("podman", "rm", "-f", "silo-abc12345-dev")
	})

	t.Run("stopped container: removes without stop", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman container exists silo-abc12345-dev":                              exec.Command("true"),
			"podman container inspect --format {{.State.Running}} silo-abc12345-dev": exec.Command("echo", "false"),
		})
		if err := cmd.DevcontainerRemove([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mock.AssertNoExec("podman", "stop", "<...>")
		mock.AssertExec("podman", "rm", "-f", "silo-abc12345-dev")
	})

	t.Run("container absent: no remove call", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{
			"podman container exists silo-abc12345-dev": exec.Command("false"),
		})
		if err := cmd.DevcontainerRemove([]string{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mock.AssertNoExec("podman", "rm", "<...>")
	})
}

func TestLoadDevcontainerInJSONMalformed(t *testing.T) {
	t.Run("malformed JSON returns error", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", base)
		siloDir := filepath.Join(base, "silo")
		if err := os.MkdirAll(siloDir, 0755); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(siloDir, "devcontainer.in.json")
		if err := os.WriteFile(path, []byte("{invalid json"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := internal.LoadDevcontainerInJSON()
		if err == nil {
			t.Error("expected error for malformed JSON")
		}
	})
}

func TestCmdDevcontainerGenerateMerge(t *testing.T) {
	t.Run("generates merged devcontainer when user provides devcontainer.in.json", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		internal.SetupUserConfig(t)

		// Write a user devcontainer.in.json with custom settings
		base := os.Getenv("XDG_CONFIG_HOME")
		userDevcontainer := filepath.Join(base, "silo", "devcontainer.in.json")
		userConfig := `{
			"customProperty": "user-value",
			"mergeProperty": "from-user"
		}`
		if err := os.WriteFile(userDevcontainer, []byte(userConfig), 0644); err != nil {
			t.Fatal(err)
		}

		if err := cmd.DevcontainerGenerate(); err != nil {
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

		// User's custom property should be preserved
		if parsed["customProperty"] != "user-value" {
			t.Errorf("expected customProperty from user config, got %v", parsed["customProperty"])
		}
		// User's mergeProperty should be preserved (overlaid by generated)
		if parsed["mergeProperty"] != "from-user" {
			t.Errorf("expected mergeProperty from user config, got %v", parsed["mergeProperty"])
		}
	})
}

func TestCmdDevcontainerGenerateExistingFile(t *testing.T) {
	t.Run("existing .devcontainer.json is not overwritten", func(t *testing.T) {
		cfg := internal.MinimalConfig("abc12345")
		internal.SetupWorkspace(t, cfg)
		internal.SetupUserConfig(t)
		// Pre-create .devcontainer.json
		existing := []byte(`{"name": "custom"}`)
		if err := os.WriteFile(".devcontainer.json", existing, 0644); err != nil {
			t.Fatal(err)
		}
		mock := internal.NewMock(t)
		mock.MockExec(map[string]*exec.Cmd{})
		if err := cmd.DevcontainerGenerate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := os.ReadFile(".devcontainer.json")
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(existing) {
			t.Errorf("expected existing file to be preserved, got %s", string(got))
		}
	})
}

func TestDeepMergeJSONMalformedInput(t *testing.T) {
	t.Run("scalar replaces object", func(t *testing.T) {
		base := map[string]any{"k": map[string]any{"x": 1.0}}
		overlay := map[string]any{"k": "string"}
		got := internal.DeepMergeJSON(base, overlay)
		if got["k"] != "string" {
			t.Errorf("expected overlay scalar to win, got %v", got["k"])
		}
	})

	t.Run("object replaces scalar", func(t *testing.T) {
		base := map[string]any{"k": "string"}
		overlay := map[string]any{"k": map[string]any{"x": 1.0}}
		got := internal.DeepMergeJSON(base, overlay)
		obj, ok := got["k"].(map[string]any)
		if !ok {
			t.Fatalf("expected merged object, got %v", got["k"])
		}
		if obj["x"] != 1.0 {
			t.Errorf("expected x=1, got %v", obj["x"])
		}
	})
}
