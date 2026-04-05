package main

import (
	"encoding/json"
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
	if !strings.Contains(s, "setup.sh /silo/setup.sh") {
		t.Errorf("expected setup.sh install in Containerfile.workspace output:\n%s", s)
	}
	if !strings.Contains(s, "--chmod=0755 setup.sh /silo/setup.sh") {
		t.Errorf("expected setup.sh install with chmod flag in Containerfile.workspace output:\n%s", s)
	}
	if strings.Contains(s, "chmod 0755 /silo/setup.sh") {
		t.Errorf("did not expect separate chmod step in Containerfile.workspace output:\n%s", s)
	}
	if strings.Contains(s, "ARG USER") {
		t.Error("Containerfile.workspace should not contain ARG USER")
	}
}

func TestRenderContainerfileBase(t *testing.T) {
	out, err := renderTemplate("Containerfile.base.tmpl", struct {
		User              string
		Home              string
		SharedVolumeMount string
	}{"alice", "/home/alice", "/silo/shared"})
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
	if !strings.Contains(s, "mkdir -p /silo/shared") {
		t.Error("expected shared volume mount path in Containerfile.base output")
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
	tc := TemplateContext{
		Image:         "silo-abc12345",
		User:          "alice",
		ContainerName: "silo-abc12345-dev",
		ContainerArgs: []string{"--cap-drop=ALL"},
	}
	out, err := renderTemplate("devcontainer.json.tmpl", tc)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"name": "silo-abc12345-dev"`) {
		t.Error("expected name in devcontainer.json")
	}
	if !strings.Contains(s, `"image": "silo-abc12345"`) {
		t.Error("expected image in devcontainer.json")
	}
	if !strings.Contains(s, `"remoteUser": "alice"`) {
		t.Error("expected remoteUser in devcontainer.json")
	}
	if !strings.Contains(s, `"--cap-drop=ALL"`) {
		t.Error("expected runArgs in devcontainer.json")
	}
	if strings.Contains(s, `"mounts"`) {
		t.Error("did not expect mounts in devcontainer.json when shared volume is unset")
	}
	if strings.Contains(s, `"postStartCommand"`) {
		t.Error("did not expect postStartCommand in devcontainer.json when shared volume is unset")
	}
	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("expected valid json, got error: %v\n%s", err, s)
	}
	if _, ok := parsed["mounts"]; ok {
		t.Error("did not expect mounts key in parsed devcontainer.json")
	}
}

func TestRenderDevcontainerJSONWithSharedVolume(t *testing.T) {
	tc := TemplateContext{
		Image:             "silo-abc12345",
		User:              "alice",
		ContainerName:     "silo-abc12345-dev",
		ContainerArgs:     []string{"--cap-drop=ALL"},
		SharedVolumeName:  "silo-shared",
		SharedVolumeMount: "/silo/shared",
		SetupScript:       "/silo/setup.sh",
	}
	out, err := renderTemplate("devcontainer.json.tmpl", tc)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"mounts": ["source=silo-shared,target=/silo/shared,type=volume,Z"]`) {
		t.Error("expected mounts in devcontainer.json")
	}
	if !strings.Contains(s, `"postStartCommand": "bash /silo/setup.sh"`) {
		t.Error("expected postStartCommand in devcontainer.json")
	}
	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("expected valid json, got error: %v\n%s", err, s)
	}
	if _, ok := parsed["mounts"]; !ok {
		t.Error("expected mounts key in parsed devcontainer.json")
	}
}

func TestNewTemplateContextDefaultSuffix(t *testing.T) {
	cfg := Config{
		General: GeneralConfig{
			User:          "alice",
			ContainerName: "silo-abc12345",
			ImageName:     "silo-abc12345",
		},
		Features: FeaturesConfig{Nested: false, SharedVolume: true},
	}
	tc := newTemplateContext(cfg)
	if tc.ContainerName != "silo-abc12345" {
		t.Fatalf("expected default container name, got %q", tc.ContainerName)
	}
	if tc.SetupScript != setupScriptPath {
		t.Fatalf("expected setup script path %q, got %q", setupScriptPath, tc.SetupScript)
	}
	if tc.SharedVolumeName == "" {
		t.Fatal("expected shared volume name to be set when feature is enabled")
	}
	joined := strings.Join(tc.ContainerArgs, " ")
	if !strings.Contains(joined, "--name silo-abc12345") {
		t.Fatalf("expected --name with default container name, got %v", tc.ContainerArgs)
	}
}

func TestNewTemplateContextWithSuffix(t *testing.T) {
	cfg := Config{
		General: GeneralConfig{
			User:          "alice",
			ContainerName: "silo-abc12345",
			ImageName:     "silo-abc12345",
		},
		Features: FeaturesConfig{Nested: false},
	}
	tc := newTemplateContext(cfg, "-dev")
	if tc.ContainerName != "silo-abc12345-dev" {
		t.Fatalf("expected suffixed container name, got %q", tc.ContainerName)
	}
	joined := strings.Join(tc.ContainerArgs, " ")
	if !strings.Contains(joined, "--name silo-abc12345-dev") {
		t.Fatalf("expected --name with suffixed container name, got %v", tc.ContainerArgs)
	}
}

func TestNewTemplateContextWithoutSharedVolume(t *testing.T) {
	cfg := Config{
		General: GeneralConfig{
			User:          "alice",
			ContainerName: "silo-abc12345",
			ImageName:     "silo-abc12345",
		},
		Features: FeaturesConfig{Nested: false, SharedVolume: false},
	}
	tc := newTemplateContext(cfg)
	if tc.SharedVolumeName != "" {
		t.Fatalf("expected empty shared volume name when feature disabled, got %q", tc.SharedVolumeName)
	}
}

func TestDetectNixSystem(t *testing.T) {
	sys := detectNixSystem()
	if sys != "x86_64-linux" && sys != "aarch64-linux" {
		t.Errorf("unexpected nix system %q", sys)
	}
}

func TestJSONTemplateFunc(t *testing.T) {
	fn := templateFuncs["json"].(func(any) (string, error))
	tests := []struct {
		input any
		want  string
	}{
		{[]string(nil), "null"},
		{[]string{}, "[]"},
		{[]string{"--cap-drop=ALL"}, `["--cap-drop=ALL"]`},
		{[]string{"a", "b"}, `["a","b"]`},
		{[]string{`quo"te`}, `["quo\"te"]`},
	}
	for _, tt := range tests {
		got, err := fn(tt.input)
		if err != nil {
			t.Fatalf("unexpected error for %v: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("json(%v) = %q, want %q", tt.input, got, tt.want)
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
