package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	buildDirMode  = 0755
	buildFileMode = 0644
)

// imageExists checks if a Podman image exists.
func imageExists(name string) bool {
	return execCommand("podman", "image", "exists", name).Run() == nil
}

// detectNixSystem returns the Nix system identifier for the current machine architecture.
func detectNixSystem() string {
	out, err := execCommand("uname", "-m").Output()
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

// buildUserImage builds the user image using home-user.nix.
func buildUserImage(tag string, tc TemplateContext) error {
	containerfile, err := renderTemplate("Containerfile.user.tmpl", tc)
	if err != nil {
		return fmt.Errorf("render Containerfile.user template: %w", err)
	}
	flakeNix, err := renderTemplate("flake.nix.tmpl", tc)
	if err != nil {
		return fmt.Errorf("render flake.nix template: %w", err)
	}

	configDir, err := userConfigDir()
	if err != nil {
		return fmt.Errorf("get user config directory: %w", err)
	}
	homeUserNix, err := os.ReadFile(filepath.Join(configDir, "home-user.nix"))
	if err != nil {
		return fmt.Errorf("read home-user.nix: %w", err)
	}

	podmanModule, err := templateFiles.ReadFile("templates/modules/podman.nix")
	if err != nil {
		return fmt.Errorf("read podman module: %w", err)
	}

	files := map[string][]byte{
		"Containerfile":            containerfile,
		"flake.nix":                flakeNix,
		"home-user.nix":            homeUserNix,
		"home-workspace-empty.nix": []byte(emptyHomeNix),
		"modules/podman.nix":       podmanModule,
	}
	if err := runBuild(tag, files); err != nil {
		return fmt.Errorf("build user image: %w", err)
	}
	return nil
}

// buildWorkspaceImage builds the workspace image layered on top of the user image.
func buildWorkspaceImage(tag string, tc TemplateContext) error {
	containerfile, err := renderTemplate("Containerfile.workspace.tmpl", tc)
	if err != nil {
		return fmt.Errorf("render Containerfile.workspace template: %w", err)
	}

	homeWorkspaceNix, err := os.ReadFile(filepath.Join(siloDir, "home.nix"))
	if err != nil {
		fallback, renderErr := renderWorkspaceHomeNix(false)
		if renderErr != nil {
			return fmt.Errorf("render workspace home.nix: %w", renderErr)
		}
		homeWorkspaceNix = []byte(fallback)
	}

	files := map[string][]byte{
		"Containerfile":      containerfile,
		"home-workspace.nix": homeWorkspaceNix,
	}
	if err := runBuild(tag, files); err != nil {
		return fmt.Errorf("build workspace image: %w", err)
	}
	return nil
}

// runBuild writes files to a temporary directory and runs podman build.
func runBuild(tag string, files map[string][]byte) error {
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

	if err := runVisible("podman", "build", "-t", tag, dir); err != nil {
		return fmt.Errorf("run podman build: %w", err)
	}
	return nil
}

// cmdBuild implements `silo build`. Builds the workspace image if missing.
func cmdBuild() error {
	cfg, _, err := ensureInit(nil)
	if err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}
	if err := ensureImages(cfg); err != nil {
		return err
	}
	return nil
}

// cmdUserBuild implements `silo user build`. Builds the user image if missing.
func cmdUserBuild() error {
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}
	cfg := Config{
		General: GeneralConfig{
			User:          u.Username,
			ContainerName: "silo-" + u.Username,
		},
	}
	tc, err := newTemplateContext(cfg)
	if err != nil {
		return fmt.Errorf("build template context: %w", err)
	}
	if imageExists(tc.BaseImage) {
		fmt.Printf("%s already exists\n", tc.BaseImage)
		return nil
	}
	if err := ensureUserImage(tc); err != nil {
		return err
	}
	return nil
}
