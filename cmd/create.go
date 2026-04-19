package cmd

import (
	"flag"
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Create creates the container from the workspace image.
func Create(args []string) error {
	flags, err := ParseCreateFlags(args)
	if err != nil {
		return fmt.Errorf("parse create flags: %w", err)
	}
	cfg, err := internal.EnsureBuilt()
	if err != nil {
		return fmt.Errorf("build images: %w", err)
	}

	extraArgs := cfg.Create.Arguments

	if flags.DryRun {
		podmanArgs, err := internal.BuildContainerArgs(cfg)
		if err != nil {
			return fmt.Errorf("build container arguments: %w", err)
		}
		createArgs := append([]string{"create"}, podmanArgs...)
		createArgs = append(createArgs, extraArgs...)
		createArgs = append(createArgs, cfg.General.ImageName)
		internal.PrintDryRun(createArgs)
		return nil
	}

	if internal.ContainerExists(cfg.General.ContainerName) {
		fmt.Printf("%s already exists\n", cfg.General.ContainerName)
		return nil
	}

	if err := internal.CreateContainer(cfg, extraArgs); err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	return nil
}

// Start implements `silo start`.
func Start() error {
	_, err := internal.EnsureStarted()
	return err
}

// CreateFlags holds parsed flags for the create command.
type CreateFlags struct {
	DryRun bool
}

// ParseCreateFlags parses the flags for `silo create`.
func ParseCreateFlags(args []string) (CreateFlags, error) {
	fs := flag.NewFlagSet("silo create", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "Print the podman create command without running it")
	fs.Usage = func() {}
	if err := fs.Parse(args); err != nil {
		return CreateFlags{}, fmt.Errorf("parse create flags: %w", err)
	}
	return CreateFlags{DryRun: *dryRun}, nil
}
