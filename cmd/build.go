package cmd

import (
	"fmt"
	"os/user"

	"github.com/mrkuz/silo/internal"
)

// Build implements `silo build`. Builds the workspace image if missing.
func Build(args []string) error {
	force, err := parseForceFlag(args, "silo build", "Force rebuild workspace image", "parse build flags")
	if err != nil {
		return err
	}
	cfg, _, err := internal.EnsureInit(nil, nil)
	if err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}
	if !force && internal.ImageExists(cfg.General.ImageName) {
		fmt.Printf("%s already exists\n", cfg.General.ImageName)
		return nil
	}
	if internal.ContainerRunning(cfg.General.ContainerName) {
		return fmt.Errorf("container %s is running", cfg.General.ContainerName)
	}
	if internal.ContainerExists(cfg.General.ContainerName) {
		return fmt.Errorf("container %s exists", cfg.General.ContainerName)
	}
	if err := internal.EnsureImages(cfg, force); err != nil {
		return err
	}
	return nil
}

// UserBuild implements `silo user build`. Builds the user image if missing.
func UserBuild(args []string) error {
	force, err := parseForceFlag(args, "silo user build", "Force rebuild user image", "parse user build flags")
	if err != nil {
		return err
	}
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}
	cfg := internal.Config{
		General: internal.GeneralConfig{
			User:          u.Username,
			ContainerName: "silo-" + u.Username,
		},
	}
	tc, err := internal.NewTemplateContext(cfg)
	if err != nil {
		return fmt.Errorf("build template context: %w", err)
	}
	if err := internal.EnsureUserFiles(false); err != nil {
		return fmt.Errorf("ensure user files: %w", err)
	}
	if !force && internal.ImageExists(tc.BaseImage) {
		fmt.Printf("%s already exists\n", tc.BaseImage)
		return nil
	}
	if err := internal.EnsureUserImage(tc, force); err != nil {
		return err
	}
	return nil
}
