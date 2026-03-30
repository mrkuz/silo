package main

import (
	"reflect"
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
