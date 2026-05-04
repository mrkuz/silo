package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Remove implements `silo rm [--force]`.
func Remove(args []string) error {
	force, err := ParseRemoveFlags("rm", args)
	if err != nil {
		return err
	}
	cfg, err := internal.RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	containerName := internal.WorkspaceContainerName(cfg.General.ID)
	imageName := internal.WorkspaceImageName(cfg.General.ID)
	if !force && internal.ContainerExists(containerName) && internal.ContainerRunning(containerName) {
		return fmt.Errorf("%s is running", containerName)
	}
	if force && internal.ContainerExists(containerName) {
		if internal.ContainerRunning(containerName) {
			if err := internal.StopContainer(containerName); err != nil {
				return fmt.Errorf("stop container: %w", err)
			}
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
		internal.PrintNotFound(imageName)
	}
	return nil
}

// ParseRemoveFlags parses the flags for `silo rm`.
func ParseRemoveFlags(cmdName string, args []string) (bool, error) {
	fs := NewFlagSet("silo rm")
	force := fs.Bool("force", false, "Stop and remove the container before removing the image")
	forceShort := fs.Bool("f", false, "")
	if err := parseWithInterceptor(fs, args); err != nil {
		return false, err
	}
	if len(fs.Args()) > 0 {
		return false, ErroneousCommand()
	}
	return *force || *forceShort, nil
}