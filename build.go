package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	containerfile, err := renderTemplate("Containerfile.base.tmpl", tc)
	if err != nil {
		return fmt.Errorf("render Containerfile.base template: %w", err)
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

	files := map[string][]byte{
		"Containerfile":            containerfile,
		"flake.nix":                flakeNix,
		"home-user.nix":            homeUserNix,
		"home-workspace-empty.nix": []byte(emptyHomeNix),
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
	setupScript, err := renderTemplate("setup.sh.tmpl", tc)
	if err != nil {
		return fmt.Errorf("render setup.sh template: %w", err)
	}

	homeWorkspaceNix, err := os.ReadFile(filepath.Join(siloDir, "home.nix"))
	if err != nil {
		homeWorkspaceNix = []byte(emptyHomeNix)
	}

	files := map[string][]byte{
		"Containerfile":      containerfile,
		"home-workspace.nix": homeWorkspaceNix,
		"setup.sh":           setupScript,
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
		if err := os.WriteFile(filepath.Join(dir, name), content, 0644); err != nil {
			return fmt.Errorf("write file to build directory: %w", err)
		}
	}

	if err := runVisible("podman", "build", "-t", tag, dir); err != nil {
		return fmt.Errorf("run podman build: %w", err)
	}
	return nil
}

// cmdBuild implements `silo build [--user]`.
func cmdBuild(args []string) error {
	flags, err := parseBuildFlags(args)
	if err != nil {
		return fmt.Errorf("parse build flags: %w", err)
	}

	cfg, err := ensureInit()
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	tc, err := newTemplateContext(cfg)
	if err != nil {
		return fmt.Errorf("build template context: %w", err)
	}
	userTag := tc.BaseImage
	wsTag := cfg.General.ImageName

	if flags.user {
		if imageExists(userTag) {
			fmt.Printf("%s already exists\n", userTag)
			return nil
		}
		fmt.Printf("Building user image %s...\n", userTag)
		if err := buildUserImage(userTag, tc); err != nil {
			return fmt.Errorf("build user image: %w", err)
		}
		return nil
	}

	if imageExists(wsTag) {
		fmt.Printf("%s already exists\n", wsTag)
		return nil
	}
	fmt.Printf("Building workspace image %s...\n", wsTag)
	if err := buildWorkspaceImage(wsTag, tc); err != nil {
		return fmt.Errorf("build workspace image: %w", err)
	}
	return nil
}

type buildFlags struct {
	user bool
}

func parseBuildFlags(args []string) (buildFlags, error) {
	fs := flag.NewFlagSet("silo build", flag.ContinueOnError)
	user := fs.Bool("user", false, "Build only the user image")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return buildFlags{}, fmt.Errorf("parse build flags: %w", err)
	}
	return buildFlags{user: *user}, nil
}
