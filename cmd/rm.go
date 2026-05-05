package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Remove implements `silo rm`.
func Remove() error {
	cfg, err := internal.RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	imageName := internal.WorkspaceImageName(cfg.General.ID)
	containerName := internal.WorkspaceContainerName(cfg.General.ID)
	if internal.ContainerExists(containerName) {
		if internal.ContainerRunning(containerName) {
			return fmt.Errorf("%s is running", containerName)
		}
		fmt.Printf("Removing %s...\n", containerName)
		if err := internal.RemoveContainer(containerName); err != nil {
			return fmt.Errorf("remove container: %w", err)
		}
	}
	if internal.ImageExists(imageName) {
		fmt.Printf("Removing %s...\n", imageName)
		if err := internal.RemoveImage(imageName); err != nil {
			return fmt.Errorf("remove image: %w", err)
		}
	} else {
		internal.PrintNotFound(imageName)
	}
	return nil
}