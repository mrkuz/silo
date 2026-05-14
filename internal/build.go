package internal

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	buildDirMode  = 0755
	buildFileMode = 0644
)

// ImageExists checks if a Podman image exists.
func ImageExists(name string) bool {
	return ExecCommand("podman", "image", "exists", name).Run() == nil
}

// RemoveImage removes the named image.
func RemoveImage(name string) error {
	if err := RunVisible("podman", "rmi", name); err != nil {
		return fmt.Errorf("remove image: %w", err)
	}
	return nil
}

// BuildImage builds the workspace image with user and workspace config baked in.
func BuildImage(tag string, tc TemplateContext, noCache bool) error {
	containerfile, err := RenderTemplate("Containerfile.tmpl", tc)
	if err != nil {
		return fmt.Errorf("render Containerfile template: %w", err)
	}
	flakeNix, err := RenderTemplate("flake.nix.tmpl", tc)
	if err != nil {
		return fmt.Errorf("render flake.nix template: %w", err)
	}

	configDir, err := UserConfigDir()
	if err != nil {
		return fmt.Errorf("get user config directory: %w", err)
	}
	homeUserNix, err := ReadFile(filepath.Join(configDir, "home.user.nix"))
	if err != nil {
		return fmt.Errorf("read home.user.nix: %w", err)
	}

	podmanModule, err := ReadTemplate("modules/podman.nix")
	if err != nil {
		return fmt.Errorf("read podman module: %w", err)
	}
	siloModule, err := ReadTemplate("modules/silo.nix")
	if err != nil {
		return fmt.Errorf("read silo module: %w", err)
	}

	homeWorkspaceNix, err := ReadFile(filepath.Join(SiloDir(), "home.nix"))
	if err != nil {
		return fmt.Errorf("read home.nix: %w", err)
	}

	files := map[string][]byte{
		"Containerfile":      containerfile,
		"flake.nix":          flakeNix,
		"home.user.nix":      homeUserNix,
		"home.empty.nix":     []byte(EmptyHomeNix),
		"home.nix":           homeWorkspaceNix,
		"modules/podman.nix": podmanModule,
		"modules/silo.nix":   siloModule,
	}
	if err := RunBuild(tag, files, noCache); err != nil {
		return fmt.Errorf("build image: %w", err)
	}
	return nil
}

// RunBuild writes files to a temporary directory and runs podman build.
func RunBuild(tag string, files map[string][]byte, noCache bool) error {
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

	args := []string{"build", "-t", tag}
	if noCache {
		args = append(args, "--no-cache")
	}
	args = append(args, dir)
	if err := RunVisible("podman", args...); err != nil {
		return fmt.Errorf("run podman build: %w", err)
	}
	return nil
}
