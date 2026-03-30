package main

import (
	"strings"
	"testing"
)

func TestRenderFlakeNix(t *testing.T) {
	out, err := renderTemplate("flake.nix.tmpl", struct {
		User   string
		System string
	}{"alice", "x86_64-linux"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `user = "alice"`) {
		t.Errorf("expected user = alice in flake.nix output:\n%s", s)
	}
	if !strings.Contains(s, `system = "x86_64-linux"`) {
		t.Errorf("expected system = x86_64-linux in flake.nix output:\n%s", s)
	}
}

func TestRenderFlakeNixAarch64(t *testing.T) {
	out, err := renderTemplate("flake.nix.tmpl", struct {
		User   string
		System string
	}{"bob", "aarch64-linux"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `user = "bob"`) {
		t.Error("expected user = bob")
	}
	if !strings.Contains(string(out), `system = "aarch64-linux"`) {
		t.Error("expected system = aarch64-linux")
	}
}

func TestRenderContainerfileWorkspace(t *testing.T) {
	out, err := renderTemplate("Containerfile.workspace.tmpl", struct {
		BaseImage string
		User      string
		Home      string
	}{"silo-alice", "alice", "/home/alice"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "FROM silo-alice") {
		t.Errorf("expected FROM silo-alice in Containerfile.workspace output:\n%s", s)
	}
	if strings.Contains(s, "ARG USER") {
		t.Error("Containerfile.workspace should not contain ARG USER")
	}
}

func TestRenderContainerfileBase(t *testing.T) {
	out, err := renderTemplate("Containerfile.base.tmpl", struct {
		User string
		Home string
	}{"alice", "/home/alice"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "FROM alpine") {
		t.Error("expected FROM alpine in Containerfile.base output")
	}
	if !strings.Contains(s, "alice") {
		t.Error("expected user alice in Containerfile.base output")
	}
	if !strings.Contains(s, "/home/alice") {
		t.Error("expected home /home/alice in Containerfile.base output")
	}
	if strings.Contains(s, "ARG USER") {
		t.Error("Containerfile.base should not contain ARG USER")
	}
}

func TestHomeEmptyNixConstant(t *testing.T) {
	if len(emptyHomeNix) == 0 {
		t.Error("emptyHomeNix constant should not be empty")
	}
	if !strings.Contains(emptyHomeNix, "pkgs") {
		t.Error("emptyHomeNix should contain pkgs argument")
	}
}

func TestRenderDevcontainerJSON(t *testing.T) {
	out, err := renderTemplate("devcontainer.json.tmpl", struct {
		Image   string
		User    string
		RunArgs string
	}{"silo-abc12345", "alice", `["--cap-drop=ALL"]`})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"image": "silo-abc12345"`) {
		t.Error("expected image in devcontainer.json")
	}
	if !strings.Contains(s, `"remoteUser": "alice"`) {
		t.Error("expected remoteUser in devcontainer.json")
	}
	if !strings.Contains(s, `["--cap-drop=ALL"]`) {
		t.Error("expected runArgs in devcontainer.json")
	}
}

func TestDetectNixSystem(t *testing.T) {
	sys := detectNixSystem()
	if sys != "x86_64-linux" && sys != "aarch64-linux" {
		t.Errorf("unexpected nix system %q", sys)
	}
}

func TestJSONStringArray(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{nil, "[]"},
		{[]string{}, "[]"},
		{[]string{"--cap-drop=ALL"}, `["--cap-drop=ALL"]`},
		{[]string{"a", "b"}, `["a", "b"]`},
		{[]string{`quo"te`}, `["quo\"te"]`},
	}
	for _, tt := range tests {
		got := jsonStringArray(tt.input)
		if got != tt.want {
			t.Errorf("jsonStringArray(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRenderTemplateError(t *testing.T) {
	_, err := renderTemplate("nonexistent.tmpl", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent template, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent.tmpl") {
		t.Errorf("expected template name in error, got: %v", err)
	}
}
