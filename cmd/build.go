package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Build implements `silo build`. Builds the workspace image if missing.
func Build(args []string) error {
	flags, err := ParseRebuildAndNoCache(args)
	if err != nil {
		return err
	}
	cfg, err := internal.EnsureInit(nil)
	if err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}
	imageName := internal.WorkspaceImageName(cfg.General.ID)
	containerName := internal.WorkspaceContainerName(cfg.General.ID)

	if !flags.Rebuild && internal.ImageExists(imageName) {
		fmt.Printf("%s already exists\n", imageName)
		return nil
	}
	if internal.ContainerRunning(containerName) {
		return fmt.Errorf("container %s is running", containerName)
	}
	if internal.ContainerExists(containerName) {
		return fmt.Errorf("container %s exists", containerName)
	}

	mergedCfg, err := internal.RequireMergedConfig()
	if err != nil {
		return fmt.Errorf("load merged config: %w", err)
	}
	tc, err := internal.NewTemplateContext(mergedCfg)
	if err != nil {
		return fmt.Errorf("build template context: %w", err)
	}

	if flags.Rebuild && internal.ImageExists(imageName) {
		if err := internal.RemoveImage(imageName); err != nil {
			return fmt.Errorf("remove existing image: %w", err)
		}
	}
	fmt.Printf("Building workspace image %s...\n", imageName)
	if err := internal.BuildImage(imageName, tc, flags.NoCache); err != nil {
		return fmt.Errorf("build workspace image: %w", err)
	}
	return nil
}

// RebuildAndNoCacheFlags holds parsed flags for build.
type RebuildAndNoCacheFlags struct {
	Rebuild bool
	NoCache bool
}

// ParseRebuildAndNoCache parses --rebuild and --no-cache flags.
func ParseRebuildAndNoCache(args []string) (RebuildAndNoCacheFlags, error) {
	fs := NewFlagSet("silo build")
	rebuildFlag := fs.Bool("rebuild", false, "")
	noCacheFlag := fs.Bool("no-cache", false, "")
	if err := parseWithInterceptor(fs, args); err != nil {
		return RebuildAndNoCacheFlags{}, err
	}
	if len(fs.Args()) > 0 {
		return RebuildAndNoCacheFlags{}, ErroneousCommand()
	}
	return RebuildAndNoCacheFlags{
		Rebuild: *rebuildFlag,
		NoCache: *noCacheFlag,
	}, nil
}
