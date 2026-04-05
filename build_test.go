package main

import (
	"os/exec"
	"testing"
)

func TestImageExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		calls := mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-test": exec.Command("true"),
		})
		if !imageExists("silo-test") {
			t.Error("expected imageExists to return true")
		}
		if !anyCall(calls, "podman", "image", "exists", "silo-test") {
			t.Errorf("expected podman image exists call, got %v", *calls)
		}
	})

	t.Run("not exists", func(t *testing.T) {
		mockExecCommand(t, map[string]*exec.Cmd{
			"podman image exists silo-test": exec.Command("false"),
		})
		if imageExists("silo-test") {
			t.Error("expected imageExists to return false")
		}
	})
}

func TestParseBuildFlags(t *testing.T) {
	tests := []struct {
		args      []string
		wantBase  bool
		wantForce bool
		wantErr   bool
	}{
		{[]string{}, false, false, false},
		{[]string{"--base"}, true, false, false},
		{[]string{"--force"}, false, true, false},
		{[]string{"--base", "--force"}, true, true, false},
		{[]string{"-f"}, false, true, false},
		{[]string{"--unknown"}, false, false, true},
	}
	for _, tt := range tests {
		f, err := parseBuildFlags(tt.args)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseBuildFlags(%v): expected error", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseBuildFlags(%v): unexpected error: %v", tt.args, err)
			continue
		}
		if f.base != tt.wantBase || f.force != tt.wantForce {
			t.Errorf("parseBuildFlags(%v) = {base:%v force:%v}, want {base:%v force:%v}",
				tt.args, f.base, f.force, tt.wantBase, tt.wantForce)
		}
	}
}
