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

	// Pass podman flag directly to EnsureInit so it can preserve seeded value when nil
	_, _, err = internal.EnsureInit(flags.Podman)
	if err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}

	// Apply shared-volume flag if explicitly set (podman is handled inside EnsureInit)
	if flags.SharedVolume != nil {
		cfg, err := internal.ParseTOML(internal.SiloToml())
		if err != nil {
			return fmt.Errorf("reload config after init: %w", err)
		}
		if cfg.Features.SharedVolume != *flags.SharedVolume {
			cfg.Features.SharedVolume = *flags.SharedVolume
			if err := cfg.SaveWorkspaceConfig(); err != nil {
				return fmt.Errorf("save silo.toml: %w", err)
			}
		}
	}
	return nil
}

// UserInit implements `silo user init`. Prints per-file status
// (for existing and new files) and delegates the actual
// file creation to EnsureUserFiles.
func UserInit() error {
	files, err := internal.UserStarterFiles()
	if err != nil {
		return fmt.Errorf("list user starter files: %w", err)
	}
	for _, f := range files {
		if err := internal.PrintInitFileStatus(f.Path); err != nil {
			return err
		}
	}
	if err := internal.EnsureUserFiles(); err != nil {
		return fmt.Errorf("create user files: %w", err)
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
}

// ParseInitFlags parses the flags for `silo init`.
func ParseInitFlags(args []string) (InitFlags, error) {
	fs := flag.NewFlagSet("silo init", flag.ContinueOnError)
	podman := fs.Bool("podman", false, "Enable Podman inside the container")
	noPodman := fs.Bool("no-podman", false, "Disable Podman inside the container")
	sharedVolume := fs.Bool("shared-volume", false, "Enable shared volume")
	noSharedVolume := fs.Bool("no-shared-volume", false, "Disable shared volume")
	fs.Usage = func() {}
	if err := fs.Parse(args); err != nil {
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
	}, nil
}

// WithoutArgs wraps a no-argument function to match the command signature.
func WithoutArgs(f func() error) func([]string) error {
	return func(_ []string) error { return f() }
}
