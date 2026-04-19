package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderFlakeNix(t *testing.T) {
	out, err := RenderTemplate("flake.nix.tmpl", struct {
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
	out, err := RenderTemplate("flake.nix.tmpl", struct {
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
	out, err := RenderTemplate("Containerfile.workspace.tmpl", struct {
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
	if !strings.Contains(s, "home-manager switch") {
		t.Errorf("expected home-manager switch in Containerfile.workspace output:\n%s", s)
	}
	if strings.Contains(s, "setup.sh") {
		t.Errorf("did not expect setup.sh in Containerfile.workspace output:\n%s", s)
	}
}

func TestRenderContainerfileUser(t *testing.T) {
	out, err := RenderTemplate("Containerfile.user.tmpl", struct {
		User              string
		Home              string
		SharedVolumeMount string
	}{"alice", "/home/alice", "/silo/shared"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "FROM alpine") {
		t.Error("expected FROM alpine in Containerfile.user output")
	}
	if !strings.Contains(s, "alice") {
		t.Error("expected user alice in Containerfile.user output")
	}
	if !strings.Contains(s, "/home/alice") {
		t.Error("expected home /home/alice in Containerfile.user output")
	}
	if !strings.Contains(s, "mkdir -p /silo/shared") {
		t.Error("expected shared volume mount path in Containerfile.user output")
	}
	if strings.Contains(s, "ARG USER") {
		t.Error("Containerfile.user should not contain ARG USER")
	}
}

func TestHomeEmptyNixConstant(t *testing.T) {
	if len(WorkspaceHomeNixTmpl) == 0 {
		t.Error("WorkspaceHomeNixTmpl constant should not be empty")
	}
	if !strings.Contains(WorkspaceHomeNixTmpl, "pkgs") {
		t.Error("WorkspaceHomeNixTmpl should contain pkgs argument")
	}
}

func TestRenderWorkspaceHomeNix(t *testing.T) {
	content, err := RenderWorkspaceHomeNix(true)
	if err != nil {
		t.Fatalf("RenderWorkspaceHomeNix(true) failed: %v", err)
	}
	if !strings.Contains(content, "module.podman.enable = true") {
		t.Errorf("expected 'module.podman.enable = true' in output, got: %s", content)
	}

	content, err = RenderWorkspaceHomeNix(false)
	if err != nil {
		t.Fatalf("RenderWorkspaceHomeNix(false) failed: %v", err)
	}
	if !strings.Contains(content, "module.podman.enable = false") {
		t.Errorf("expected 'module.podman.enable = false' in output, got: %s", content)
	}
}

func TestRenderDevcontainerJSON(t *testing.T) {
	tc := TemplateContext{
		Image:            "silo-abc12345",
		User:             "alice",
		ContainerName:    "silo-abc12345-dev",
		DevcontainerArgs: []string{"--name", "silo-abc12345-dev", "--hostname", "silo-abc12345-dev", "--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges"},
	}
	out, err := RenderTemplate("devcontainer.json.tmpl", tc)
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
		Image:            "silo-abc12345",
		User:             "alice",
		ContainerName:    "silo-abc12345-dev",
		DevcontainerArgs: []string{"--name", "silo-abc12345-dev", "--hostname", "silo-abc12345-dev", "--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges"},
		SharedVolumeName: "silo-shared",
		SharedVolumePaths: []string{"/home/alice/.cache/uv", "/home/alice/.config/nvim"},
	}
	out, err := RenderTemplate("devcontainer.json.tmpl", tc)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"type=volume,source=silo-shared,target=/home/alice/.cache/uv,subpath=home/alice/.cache/uv,Z"`) {
		t.Error("expected volume mount with subpath for .cache/uv in devcontainer.json")
	}
	if !strings.Contains(s, `"type=volume,source=silo-shared,target=/home/alice/.config/nvim,subpath=home/alice/.config/nvim,Z"`) {
		t.Error("expected volume mount with subpath for .config/nvim in devcontainer.json")
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
		Features: FeaturesConfig{Podman: false, SharedVolume: true},
		SharedVolume: SharedVolumeConfig{
			Name:  "silo-shared",
			Paths: []string{"$HOME/.cache/uv/", "$HOME/.config/nvim/"},
		},
	}
	tc, err := NewTemplateContext(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tc.ContainerName != "silo-abc12345" {
		t.Fatalf("expected default container name, got %q", tc.ContainerName)
	}
	if tc.SharedVolumeName == "" {
		t.Fatal("expected shared volume name to be set when feature is enabled")
	}
	if len(tc.SharedVolumePaths) != 2 {
		t.Fatalf("expected 2 shared volume paths, got %d", len(tc.SharedVolumePaths))
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
		Features: FeaturesConfig{Podman: false},
	}
	tc, err := NewTemplateContext(cfg, "-dev")
	if err != nil {
		t.Fatal(err)
	}
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
		Features: FeaturesConfig{Podman: false, SharedVolume: false},
	}
	tc, err := NewTemplateContext(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tc.SharedVolumeName != "" {
		t.Fatalf("expected empty shared volume name when feature disabled, got %q", tc.SharedVolumeName)
	}
}

func TestNewTemplateContextWorkspaceMount(t *testing.T) {
	cfg := Config{
		General: GeneralConfig{
			ID:            "abc12345",
			User:          "alice",
			ContainerName: "silo-abc12345",
			ImageName:     "silo-abc12345",
		},
		Features: FeaturesConfig{Podman: false},
	}
	tc, err := NewTemplateContext(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	want := "/workspace/abc12345/" + filepath.Base(cwd)
	if tc.WorkspaceMount != want {
		t.Errorf("WorkspaceMount = %q, want %q", tc.WorkspaceMount, want)
	}
}

func TestRenderDevcontainerJSONWorkspaceMount(t *testing.T) {
	tc := TemplateContext{
		Image:            "silo-abc12345",
		User:             "alice",
		ContainerName:    "silo-abc12345-dev",
		DevcontainerArgs: []string{"--name", "silo-abc12345-dev", "--hostname", "silo-abc12345-dev", "--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges"},
		WorkspaceMount:   "/workspace/abc12345/myproject",
	}
	out, err := RenderTemplate("devcontainer.json.tmpl", tc)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("expected valid json, got error: %v\n%s", err, out)
	}
	if got := parsed["workspaceFolder"]; got != "/workspace/abc12345/myproject" {
		t.Errorf("workspaceFolder = %q, want %q", got, "/workspace/abc12345/myproject")
	}
	wantMount := "source=${localWorkspaceFolder},target=/workspace/abc12345/myproject,type=bind,z"
	if got := parsed["workspaceMount"]; got != wantMount {
		t.Errorf("workspaceMount = %q, want %q", got, wantMount)
	}
}

func TestDetectNixSystem(t *testing.T) {
	sys := DetectNixSystem()
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
		{[]string{"quo\"te"}, `["quo\"te"]`},
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
	_, err := RenderTemplate("nonexistent.tmpl", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent template, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent.tmpl") {
		t.Errorf("expected template name in error, got: %v", err)
	}
}
