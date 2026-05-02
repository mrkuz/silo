package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// VolumeSetup creates directories on the shared volume so they can be mounted as subpath volumes.
func VolumeSetup() error {
	cfg, err := internal.RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	performed, err := internal.VolumeSetup(cfg)
	if err != nil {
		return fmt.Errorf("volume setup: %w", err)
	}
	if performed {
		fmt.Println("volume setup complete")
	}
	return nil
}