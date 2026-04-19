package cmd

import (
	"flag"
	"fmt"
	"os/user"

	"github.com/mrkuz/silo/internal"
)

// Stop implements `silo stop`.
func Stop() error {
	cfg, err := internal.RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	if !internal.ContainerRunning(cfg.General.ContainerName) {
		fmt.Printf("%s is not running\n", cfg.General.ContainerName)
		return nil
	}
	if err := internal.StopContainer(cfg.General.ContainerName); err != nil {
		return fmt.Errorf("stop container: %w", err)
	}
	return nil
}

// Status implements `silo status`.
func Status() error {
	cfg, err := internal.RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	internal.PrintRunningStatus(internal.ContainerRunning(cfg.General.ContainerName))
	return nil
}

// Remove removes the workspace container.
func Remove(args []string) error {
	force, err := ParseRemoveFlags(args)
	if err != nil {
		return fmt.Errorf("parse remove flags: %w", err)
	}
	cfg, err := internal.RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	if err := internal.RemoveNamedContainer(cfg.General.ContainerName, force); err != nil {
		return err
	}
	return nil
}

// RemoveImage implements `silo rmi [--force]`.
func RemoveImage(args []string) error {
	force, err := ParseRemoveImageFlags(args)
	if err != nil {
		return fmt.Errorf("parse rmi flags: %w", err)
	}
	cfg, err := internal.RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	if force && internal.ContainerExists(cfg.General.ContainerName) {
		if internal.ContainerRunning(cfg.General.ContainerName) {
			if err := internal.StopContainer(cfg.General.ContainerName); err != nil {
				return fmt.Errorf("stop container: %w", err)
			}
		}
		fmt.Printf("Removing %s...\n", cfg.General.ContainerName)
		if err := internal.RemoveContainer(cfg.General.ContainerName); err != nil {
			return fmt.Errorf("remove container: %w", err)
		}
	}
	if internal.ImageExists(cfg.General.ImageName) {
		fmt.Printf("Removing %s...\n", cfg.General.ImageName)
		if err := internal.RemoveImage(cfg.General.ImageName); err != nil {
			return fmt.Errorf("remove image: %w", err)
		}
	} else {
		internal.PrintNotFound(cfg.General.ImageName)
	}
	return nil
}

// UserRmi implements `silo user rmi`. Removes the user image.
func UserRmi() error {
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}
	userImage := internal.BaseImageName(u.Username)
	if internal.ImageExists(userImage) {
		fmt.Printf("Removing %s...\n", userImage)
		if err := internal.RemoveImage(userImage); err != nil {
			return fmt.Errorf("remove user image: %w", err)
		}
	} else {
		internal.PrintNotFound(userImage)
	}
	return nil
}

// ParseRemoveFlags parses the flags for `silo rm`.
func ParseRemoveFlags(args []string) (bool, error) {
	return parseForceFlag(args, "silo rm", "Stop and remove a running container", "parse remove flags")
}

// ParseRemoveImageFlags parses the flags for `silo rmi`.
func ParseRemoveImageFlags(args []string) (bool, error) {
	return parseForceFlag(args, "silo rmi", "Stop and remove the container before removing the image", "parse rmi flags")
}

func parseForceFlag(args []string, name, usage, context string) (bool, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	force := fs.Bool("force", false, usage)
	forceShort := fs.Bool("f", false, "")
	fs.Usage = func() {}
	if err := fs.Parse(args); err != nil {
		return false, fmt.Errorf("%s: %w", context, err)
	}
	return *force || *forceShort, nil
}
