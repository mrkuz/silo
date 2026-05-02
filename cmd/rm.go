package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Remove implements `silo rm [--force]`.
func Remove(args []string) error {
	force, err := ParseRemoveFlags(args)
	if err != nil {
		return fmt.Errorf("parse rm flags: %w", err)
	}
	cfg, err := internal.RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	if !force && internal.ContainerExists(cfg.General.ContainerName) && internal.ContainerRunning(cfg.General.ContainerName) {
		return fmt.Errorf("%s is running", cfg.General.ContainerName)
	}
	if force && internal.ContainerExists(cfg.General.ContainerName) {
		if internal.ContainerRunning(cfg.General.ContainerName) {
			if err := internal.StopContainer(cfg.General.ContainerName); err != nil {
				return fmt.Errorf("stop container: %w", err)
			}
		}
		fmt.Printf("Removing %s...\n", cfg.General.ContainerName)
		if err := internal.RemoveContainer(cfg.General.ContainerName); err != nil {
			return fmt.Errorf("remove container: %w", err)
		}
	}
	if internal.ImageExists(cfg.General.ImageName) {
		fmt.Printf("Removing %s...\n", cfg.General.ImageName)
		if err := internal.RemoveImage(cfg.General.ImageName); err != nil {
			return fmt.Errorf("remove image: %w", err)
		}
	} else {
		internal.PrintNotFound(cfg.General.ImageName)
	}
	return nil
}