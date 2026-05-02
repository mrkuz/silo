package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Connect opens an interactive shell in the running container.
func Connect(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("connect does not take arguments")
	}
	cfg, err := internal.RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	if !internal.ContainerExists(cfg.General.ContainerName) {
		return fmt.Errorf("container %s does not exist", cfg.General.ContainerName)
	}
	if !internal.ContainerRunning(cfg.General.ContainerName) {
		return fmt.Errorf("container %s is not running", cfg.General.ContainerName)
	}
	fmt.Printf("Connecting to %s...\n", cfg.General.ContainerName)
	if err := internal.ConnectContainer(cfg.General.ContainerName, cfg.Connect.Command, nil); err != nil {
		return fmt.Errorf("connect to container: %w", err)
	}
	return nil
}