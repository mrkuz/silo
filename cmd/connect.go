package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Connect opens an interactive shell in the running container.
func Connect() error {
	cfg, err := internal.RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	containerName := internal.WorkspaceContainerName(cfg.General.ID)
	if !internal.ContainerExists(containerName) {
		return fmt.Errorf("container %s does not exist", containerName)
	}
	if !internal.ContainerRunning(containerName) {
		return fmt.Errorf("container %s is not running", containerName)
	}
	fmt.Printf("Connecting to %s...\n", containerName)
	if err := internal.ConnectContainer(containerName); err != nil {
		return fmt.Errorf("connect to container: %w", err)
	}
	return nil
}