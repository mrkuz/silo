package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// DevcontainerGenerate generates a .devcontainer.json for VS Code.
func DevcontainerGenerate(args []string) error {
	flags, err := ParseUpdateFlags(args)
	if err != nil {
		return err
	}
	return internal.DevcontainerGenerate(flags.Update)
}

// DevcontainerStop implements `silo devcontainer stop`.
func DevcontainerStop() error {
	cfg, err := internal.RequireMergedConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	name := internal.DevContainerName(cfg)
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

// DevcontainerStatus implements `silo devcontainer status`.
func DevcontainerStatus() error {
	cfg, err := internal.RequireMergedConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	name := internal.DevContainerName(cfg)
	internal.PrintRunningStatus(internal.ContainerRunning(name))
	return nil
}

// DevcontainerConnect implements `silo devcontainer connect`.
func DevcontainerConnect() error {
	cfg, err := internal.RequireMergedConfig()
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
	if err := internal.ConnectContainer(name); err != nil {
		return fmt.Errorf("connect to container: %w", err)
	}
	return nil
}

// UpdateFlags holds parsed flags for devcontainer commands.
type UpdateFlags struct {
	Update bool
}

// ParseUpdateFlags parses --update flag.
func ParseUpdateFlags(args []string) (UpdateFlags, error) {
	fs := NewFlagSet("silo devcontainer")
	updateFlag := fs.Bool("update", false, "")
	if err := parseWithInterceptor(fs, args); err != nil {
		return UpdateFlags{}, err
	}
	if len(fs.Args()) > 0 {
		return UpdateFlags{}, ErroneousCommand()
	}
	return UpdateFlags{Update: *updateFlag}, nil
}
