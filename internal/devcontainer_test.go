package internal

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDeepMergeJSON(t *testing.T) {
	t.Run("scalar override", func(t *testing.T) {
		base := map[string]any{"a": 1.0}
		overlay := map[string]any{"a": 2.0}
		got := DeepMergeJSON(base, overlay)
		if got["a"] != 2.0 {
			t.Errorf("expected a=2, got %v", got["a"])
		}
	})

	t.Run("new key from overlay", func(t *testing.T) {
		base := map[string]any{"a": 1.0}
		overlay := map[string]any{"b": 2.0}
		got := DeepMergeJSON(base, overlay)
		if got["a"] != 1.0 || got["b"] != 2.0 {
			t.Errorf("unexpected result: %v", got)
		}
	})

	t.Run("base-only key preserved", func(t *testing.T) {
		base := map[string]any{"a": 1.0, "b": 2.0}
		overlay := map[string]any{"b": 3.0}
		got := DeepMergeJSON(base, overlay)
		if got["a"] != 1.0 {
			t.Errorf("expected base key a=1 to be preserved, got %v", got["a"])
		}
	})

	t.Run("nested object merge", func(t *testing.T) {
		base := map[string]any{"obj": map[string]any{"x": 1.0}}
		overlay := map[string]any{"obj": map[string]any{"y": 2.0}}
		got := DeepMergeJSON(base, overlay)
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
		got := DeepMergeJSON(base, overlay)
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
		got := DeepMergeJSON(base, overlay)
		if got["k"] != "string" {
			t.Errorf("expected overlay scalar to win, got %v", got["k"])
		}
	})

	t.Run("does not mutate base or overlay", func(t *testing.T) {
		base := map[string]any{"a": 1.0}
		overlay := map[string]any{"a": 2.0}
		DeepMergeJSON(base, overlay)
		if base["a"] != 1.0 {
			t.Errorf("base was mutated")
		}
	})
}

// TestDeepMergeJSONMalformedInput tests edge cases of DeepMergeJSON.
// Core merge logic tested here; integration covered by features/devcontainer_test.go.
func TestDeepMergeJSONMalformedInput(t *testing.T) {
	t.Run("scalar replaces object", func(t *testing.T) {
		base := map[string]any{"k": map[string]any{"x": 1.0}}
		overlay := map[string]any{"k": "string"}
		got := DeepMergeJSON(base, overlay)
		if got["k"] != "string" {
			t.Errorf("expected overlay scalar to win, got %v", got["k"])
		}
	})

	t.Run("object replaces scalar", func(t *testing.T) {
		base := map[string]any{"k": "string"}
		overlay := map[string]any{"k": map[string]any{"x": 1.0}}
		got := DeepMergeJSON(base, overlay)
		obj, ok := got["k"].(map[string]any)
		if !ok {
			t.Fatalf("expected merged object, got %v", got["k"])
		}
		if obj["x"] != 1.0 {
			t.Errorf("expected x=1, got %v", obj["x"])
		}
	})
}

// TestLoadDevcontainerInJSONMalformed tests edge case of loading malformed JSON.
// Integration covered by features/devcontainer_test.go.
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
		_, err := LoadDevcontainerInJSON()
		if err == nil {
			t.Error("expected error for malformed JSON")
		}
	})
}