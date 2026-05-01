package cmd

import (
	"flag"
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Init implements `silo init`. Creates workspace files only.
// Use `silo user init` to create user files.
func Init(args []string) error {
	flags, err := ParseInitFlags(args)
	if err != nil {
		return fmt.Errorf("parse init flags: %w", err)
	}
	initPaths := []string{internal.SiloToml(), internal.SiloDir() + "/home.nix"}
	for _, p := range initPaths {
		if err := internal.PrintInitFileStatus(p); err != nil {
			return err
		}
	}

	if flags.Force {
		seededCfg, err := internal.SeedWorkspaceConfig()
		if err != nil {
			return fmt.Errorf("seed config from user settings: %w", err)
		}
		existingCfg, _, err := internal.InitWorkspaceConfig()
		if err != nil {
			return fmt.Errorf("load existing workspace config: %w", err)
		}
		seededCfg.General = existingCfg.General
		if flags.Podman != nil {
			seededCfg.Features.Podman = *flags.Podman
		}
		if flags.SharedVolume != nil {
			seededCfg.Features.SharedVolume = *flags.SharedVolume
		}
		seededCfg.Create.Arguments = append(seededCfg.Create.Arguments, internal.DefaultCreateArgs(seededCfg.Features.Podman)...)
		if err := internal.EnsureWorkspaceHomeNix(seededCfg.Features.Podman, true); err != nil {
			return fmt.Errorf("overwrite workspace home.nix: %w", err)
		}
		if err := seededCfg.SaveWorkspaceConfig(); err != nil {
			return fmt.Errorf("save workspace config: %w", err)
		}
	} else {
		// On first run, apply feature flags to initial config.
		// On subsequent runs, feature flags are ignored unless --force is set.
		_, _, err := internal.InitWorkspaceConfig()
		if err != nil {
			return fmt.Errorf("initialize workspace: %w", err)
		}
		if _, _, err = internal.EnsureInit(flags.Podman, flags.SharedVolume); err != nil {
			return fmt.Errorf("initialize workspace: %w", err)
		}
	}

	return nil
}

// UserInit implements `silo user init`. Prints per-file status
// (for existing and new files) and delegates the actual
// file creation to EnsureUserFiles.
func UserInit(args []string) error {
	force, _ := extractForceFlag(args)

	files, err := internal.UserStarterFiles()
	if err != nil {
		return fmt.Errorf("list user starter files: %w", err)
	}
	for _, f := range files {
		if err := internal.PrintInitFileStatus(f.Path); err != nil {
			return err
		}
	}
	if err := internal.EnsureUserFiles(force); err != nil {
		return fmt.Errorf("ensure user files: %w", err)
	}
	return nil
}

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

// InitFlags holds parsed flags for the init command.
type InitFlags struct {
	Podman       *bool
	SharedVolume *bool
	Force        bool
}

// ParseInitFlags parses the flags for `silo init`.
func ParseInitFlags(args []string) (InitFlags, error) {
	force, remaining := extractForceFlag(args)

	fs := flag.NewFlagSet("silo init", flag.ContinueOnError)
	podman := fs.Bool("podman", false, "Enable Podman inside the container")
	noPodman := fs.Bool("no-podman", false, "Disable Podman inside the container")
	sharedVolume := fs.Bool("shared-volume", false, "Enable shared volume")
	noSharedVolume := fs.Bool("no-shared-volume", false, "Disable shared volume")
	fs.Usage = func() {}
	if err := fs.Parse(remaining); err != nil {
		return InitFlags{}, fmt.Errorf("parse init flags: %w", err)
	}
	var podmanVal, svVal *bool
	if *noPodman {
		v := false
		podmanVal = &v
	} else if *podman {
		v := true
		podmanVal = &v
	}
	if *noSharedVolume {
		v := false
		svVal = &v
	} else if *sharedVolume {
		v := true
		svVal = &v
	}
	return InitFlags{
		Podman:       podmanVal,
		SharedVolume: svVal,
		Force:        force,
	}, nil
}

// WithoutArgs wraps a no-argument function to match the command signature.
func WithoutArgs(f func() error) func([]string) error {
	return func(_ []string) error { return f() }
}

// extractForceFlag extracts -f/--force from args before FlagSet parsing.
func extractForceFlag(args []string) (force bool, remaining []string) {
	for _, arg := range args {
		if arg == "-f" || arg == "--force" {
			force = true
			continue
		}
		remaining = append(remaining, arg)
	}
	return force, remaining
}
