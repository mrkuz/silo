package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// imageExists reports whether a Podman image with the given name exists.
func imageExists(name string) bool {
	return execCommand("podman", "image", "exists", name).Run() == nil
}

// detectNixSystem maps the host machine architecture to a Nix system string.
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

// buildBaseImage builds the base image (silo-USER) using the user's home.nix.
func buildBaseImage(tag string, tc TemplateContext) error {
	containerfile, err := renderTemplate("Containerfile.base.tmpl", tc)
	if err != nil {
		return err
	}
	flakeNix, err := renderTemplate("flake.nix.tmpl", tc)
	if err != nil {
		return err
	}

	configDir, err := globalConfigDir()
	if err != nil {
		return err
	}
	homeNix, err := os.ReadFile(filepath.Join(configDir, "home.nix"))
	if err != nil {
		return fmt.Errorf("read %s/home.nix: %w", configDir, err)
	}

	files := map[string][]byte{
		"Containerfile":            containerfile,
		"flake.nix":                flakeNix,
		"home.nix":                 homeNix,
		"home-workspace-empty.nix": []byte(emptyHomeNix),
	}
	return runBuild(tag, files)
}

// buildWorkspaceImage builds the workspace image (tag) on top of baseImage,
// using .silo/home.nix as the workspace overlay if present, otherwise an empty module.
func buildWorkspaceImage(tag string, tc TemplateContext) error {
	containerfile, err := renderTemplate("Containerfile.workspace.tmpl", tc)
	if err != nil {
		return err
	}

	homeWorkspaceNix, err := os.ReadFile(filepath.Join(siloDir, "home.nix"))
	if err != nil {
		homeWorkspaceNix = []byte(emptyHomeNix)
	}

	files := map[string][]byte{
		"Containerfile":      containerfile,
		"home-workspace.nix": homeWorkspaceNix,
	}
	return runBuild(tag, files)
}

// runBuild writes files to a temporary directory and runs podman build.
func runBuild(tag string, files map[string][]byte) error {
	dir, err := os.MkdirTemp("", "silo-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), content, 0644); err != nil {
			return err
		}
	}

	return runVisible("podman", "build", "-t", tag, dir)
}

// ensureImageRemoved removes the image if force is set and the image exists.
// If force is false and the image exists, it returns false to signal "already exists".
// Returns true when the caller should proceed to build.
func ensureImageRemoved(tag string, force bool) (bool, error) {
	if !imageExists(tag) {
		return true, nil
	}
	if !force {
		return false, nil
	}
	fmt.Printf("Removing image %s...\n", tag)
	return true, removeImage(tag)
}

// cmdBuild implements `silo build [--base] [--force]`.
func cmdBuild(args []string) error {
	flags, err := parseBuildFlags(args)
	if err != nil {
		return err
	}

	cfg, err := ensureInit()
	if err != nil {
		return err
	}

	tc := newTemplateContext(cfg)
	baseTag := tc.BaseImage
	wsTag := cfg.General.ImageName

	if flags.base {
		proceed, err := ensureImageRemoved(baseTag, flags.force)
		if err != nil {
			return err
		}
		if !proceed {
			fmt.Printf("Base image %s already exists.\n", baseTag)
			return nil
		}
		fmt.Printf("Building base image %s...\n", baseTag)
		if err := buildBaseImage(baseTag, tc); err != nil {
			return err
		}
		// Also rebuild the workspace image on top of the new base.
		if _, err := ensureImageRemoved(wsTag, flags.force); err != nil {
			return err
		}
		fmt.Printf("Building workspace image %s...\n", wsTag)
		return buildWorkspaceImage(wsTag, tc)
	}

	proceed, err := ensureImageRemoved(wsTag, flags.force)
	if err != nil {
		return err
	}
	if !proceed {
		fmt.Printf("Workspace image %s already exists.\n", wsTag)
		return nil
	}
	fmt.Printf("Building workspace image %s...\n", wsTag)
	return buildWorkspaceImage(wsTag, tc)
}

type buildFlags struct {
	base  bool
	force bool
}

func parseBuildFlags(args []string) (buildFlags, error) {
	fs := flag.NewFlagSet("silo build", flag.ContinueOnError)
	base := fs.Bool("base", false, "Build the base and workspace image")
	force := fs.Bool("force", false, "Remove and rebuild the image if it already exists")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return buildFlags{}, err
	}
	return buildFlags{base: *base, force: *force}, nil
}
