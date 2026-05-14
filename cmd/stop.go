package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Stop implements `silo stop`.
func Stop() error {
	cfg, err := internal.RequireMergedConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	name := internal.WorkspaceContainerName(cfg.ID)
	if !internal.ContainerExists(name) {
		fmt.Printf("%s not found\n", name)
		return nil
	}
	if internal.ContainerRunning(name) {
		if err := internal.StopContainer(name); err != nil {
			return fmt.Errorf("stop container: %w", err)
		}
	} else {
		fmt.Printf("%s is not running\n", name)
	}
	fmt.Printf("Removing %s...\n", name)
	if err := internal.RemoveContainer(name); err != nil {
		return fmt.Errorf("remove container: %w", err)
	}
	return nil
}
