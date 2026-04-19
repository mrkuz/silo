package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// DevcontainerGenerate generates a .devcontainer.json for VS Code.
func DevcontainerGenerate() error {
	return internal.DevcontainerGenerate()
}

// DevcontainerStop implements `silo devcontainer stop`.
func DevcontainerStop() error {
	return internal.DevcontainerStop()
}

// DevcontainerStatus implements `silo devcontainer status`.
func DevcontainerStatus() error {
	return internal.DevcontainerStatus()
}

// DevcontainerRemove implements `silo devcontainer rm [-f|--force]`.
func DevcontainerRemove(args []string) error {
	force, err := ParseDevcontainerRemoveFlags(args)
	if err != nil {
		return fmt.Errorf("parse devcontainer rm flags: %w", err)
	}
	return internal.DevcontainerRemove(force)
}

// ParseDevcontainerRemoveFlags parses the flags for `silo devcontainer rm`.
func ParseDevcontainerRemoveFlags(args []string) (bool, error) {
	return parseForceFlag(args, "silo devcontainer rm", "Stop and remove a running container", "parse devcontainer rm flags")
}
