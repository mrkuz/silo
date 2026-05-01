package cmd

import (
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
