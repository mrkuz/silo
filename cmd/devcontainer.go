package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// DevcontainerGenerate generates a .devcontainer.json for VS Code.
func DevcontainerGenerate(args []string) error {
	force, _ := extractForceFlag(args)
	return internal.DevcontainerGenerate(force)
}

// DevcontainerStop implements `silo devcontainer stop`.
func DevcontainerStop() error {
	return internal.DevcontainerStop()
}

// DevcontainerStatus implements `silo devcontainer status`.
func DevcontainerStatus() error {
	return internal.DevcontainerStatus()
}

// DevcontainerConnect implements `silo devcontainer connect`.
func DevcontainerConnect() error {
	cfg, err := internal.RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	name := internal.DevContainerName(cfg)
	if !internal.ContainerExists(name) {
		return fmt.Errorf("container %s does not exist", name)
	}
	if !internal.ContainerRunning(name) {
		return fmt.Errorf("container %s is not running", name)
	}
	fmt.Printf("Connecting to %s...\n", name)
	if err := internal.ConnectContainer(name, cfg.Connect.Command, nil); err != nil {
		return fmt.Errorf("connect to container: %w", err)
	}
	return nil
}
