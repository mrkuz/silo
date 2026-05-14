package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Remove implements `silo rm`.
func Remove() error {
	cfg, err := internal.RequireMergedConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	imageName := internal.WorkspaceImageName(cfg.ID)
	containerName := internal.WorkspaceContainerName(cfg.ID)
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
		fmt.Printf("%s not found\n", imageName)
	}
	return nil
}
