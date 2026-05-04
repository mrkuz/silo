package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Build implements `silo build`. Builds the workspace image if missing.
func Build(args []string) error {
	force, _, err := ParseForceFlag("build", args)
	if err != nil {
		return err
	}
	cfg, _, err := internal.EnsureInit(nil)
	if err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}
	if !force && internal.ImageExists(internal.WorkspaceImageName(cfg.General.ID)) {
		fmt.Printf("%s already exists\n", internal.WorkspaceImageName(cfg.General.ID))
		return nil
	}
	if internal.ContainerRunning(internal.WorkspaceContainerName(cfg.General.ID)) {
		return fmt.Errorf("container %s is running", internal.WorkspaceContainerName(cfg.General.ID))
	}
	if internal.ContainerExists(internal.WorkspaceContainerName(cfg.General.ID)) {
		return fmt.Errorf("container %s exists", internal.WorkspaceContainerName(cfg.General.ID))
	}
	if err := internal.EnsureImages(cfg, force); err != nil {
		return err
	}
	return nil
}
