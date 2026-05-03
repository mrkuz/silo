package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	buildDirMode  = 0755
	buildFileMode = 0644
)

// ImageExists checks if a Podman image exists.
func ImageExists(name string) bool {
	return ExecCommand("podman", "image", "exists", name).Run() == nil
}

// DetectNixSystem returns the Nix system identifier for the current machine architecture.
func DetectNixSystem() string {
	out, err := ExecCommand("uname", "-m").Output()
	if err != nil {
		return "x86_64-linux"
	}

	switch strings.TrimSpace(string(out)) {
	case "aarch64", "arm64":
		return "aarch64-linux"
	default:
		return "x86_64-linux"
	}
}

// BuildUserImage builds the user image using home-user.nix.
func BuildUserImage(tag string, tc TemplateContext) error {
	containerfile, err := RenderTemplate("Containerfile.user.tmpl", tc)
	if err != nil {
		return fmt.Errorf("render Containerfile.user template: %w", err)
	}
	flakeNix, err := RenderTemplate("flake.nix.tmpl", tc)
	if err != nil {
		return fmt.Errorf("render flake.nix template: %w", err)
	}

	configDir, err := UserConfigDir()
	if err != nil {
		return fmt.Errorf("get user config directory: %w", err)
	}
	homeUserNix, err := ReadFile(filepath.Join(configDir, "home-user.nix"))
	if err != nil {
		return fmt.Errorf("read home-user.nix: %w", err)
	}

	podmanModule, err := ReadTemplate("modules/podman.nix")
	if err != nil {
		return fmt.Errorf("read podman module: %w", err)
	}
	siloModule, err := ReadTemplate("modules/silo.nix")
	if err != nil {
		return fmt.Errorf("read silo module: %w", err)
	}

	files := map[string][]byte{
		"Containerfile":            containerfile,
		"flake.nix":                flakeNix,
		"home-user.nix":            homeUserNix,
		"home-workspace-empty.nix": []byte(HomeUserNix),
		"modules/podman.nix":       podmanModule,
		"modules/silo.nix":         siloModule,
	}
	if err := RunBuild(tag, files); err != nil {
		return fmt.Errorf("build user image: %w", err)
	}
	return nil
}

// BuildWorkspaceImage builds the workspace image layered on top of the user image.
func BuildWorkspaceImage(tag string, tc TemplateContext) error {
	containerfile, err := RenderTemplate("Containerfile.workspace.tmpl", tc)
	if err != nil {
		return fmt.Errorf("render Containerfile.workspace template: %w", err)
	}

	homeWorkspaceNix, err := ReadFile(filepath.Join(SiloDir(), "home.nix"))
	if err != nil {
		fallback, renderErr := RenderWorkspaceHomeNix(false)
		if renderErr != nil {
			return fmt.Errorf("render workspace home.nix: %w", renderErr)
		}
		homeWorkspaceNix = []byte(fallback)
	}

	files := map[string][]byte{
		"Containerfile":      containerfile,
		"home-workspace.nix": homeWorkspaceNix,
	}
	if err := RunBuild(tag, files); err != nil {
		return fmt.Errorf("build workspace image: %w", err)
	}
	return nil
}

// RunBuild writes files to a temporary directory and runs podman build.
func RunBuild(tag string, files map[string][]byte) error {
	dir, err := os.MkdirTemp("", "silo-build-*")
	if err != nil {
		return fmt.Errorf("create temporary build directory: %w", err)
	}
	defer os.RemoveAll(dir)

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), buildDirMode); err != nil {
			return fmt.Errorf("create directory for build file: %w", err)
		}
		if err := os.WriteFile(path, content, buildFileMode); err != nil {
			return fmt.Errorf("write file to build directory: %w", err)
		}
	}

	if err := RunVisible("podman", "build", "-t", tag, dir); err != nil {
		return fmt.Errorf("run podman build: %w", err)
	}
	return nil
}

