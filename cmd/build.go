package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Build implements `silo build`. Builds the workspace image if missing.
func Build(args []string) error {
	force, _ := ParseForceFlag(args)
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
