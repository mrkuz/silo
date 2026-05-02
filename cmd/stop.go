package cmd

import (
	"flag"
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Stop implements `silo stop`.
func Stop() error {
	cfg, err := internal.RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	name := internal.WorkspaceContainerName(cfg.General.ID)
	if !internal.ContainerExists(name) {
		internal.PrintNotFound(name)
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

// ParseRemoveFlags parses the flags for `silo rm`.
func ParseRemoveFlags(args []string) (bool, error) {
	fs := flag.NewFlagSet("silo rm", flag.ContinueOnError)
	force := fs.Bool("force", false, "Stop and remove the container before removing the image")
	forceShort := fs.Bool("f", false, "")
	fs.Usage = func() {}
	if err := fs.Parse(args); err != nil {
		return false, fmt.Errorf("parse rm flags: %w", err)
	}
	return *force || *forceShort, nil
}
