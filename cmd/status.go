package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Status implements `silo status`.
func Status() error {
	cfg, err := internal.RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	internal.PrintRunningStatus(internal.ContainerRunning(cfg.General.ContainerName))
	return nil
}