package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Init implements `silo init`. Creates workspace files only.
// Use `silo user init` to create user files.
func Init(args []string) error {
	flags, err := ParseInitFlags(args)
	if err != nil {
		return err
	}
	initPaths := []string{internal.SiloToml(), internal.SiloDir() + "/home.nix"}
	for _, p := range initPaths {
		if err := internal.PrintInitFileStatus(p); err != nil {
			return err
		}
	}

	// On first run, apply feature flags to initial config.
	// On subsequent runs, feature flags are ignored.
	_, _, err = internal.InitWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}
	if _, _, err = internal.EnsureInit(flags.Podman); err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}

	return nil
}

// InitFlags holds parsed flags for the init command.
type InitFlags struct {
	Podman *bool
}

// ParseInitFlags parses the flags for `silo init`.
func ParseInitFlags(args []string) (InitFlags, error) {
	fs := NewFlagSet("silo init")
	podman := fs.Bool("podman", false, "Enable Podman inside the container")
	noPodman := fs.Bool("no-podman", false, "Disable Podman inside the container")
	if err := parseWithInterceptor(fs, args); err != nil {
		return InitFlags{}, err
	}
	if len(fs.Args()) > 0 {
		return InitFlags{}, ErroneousCommand()
	}
	var podmanVal *bool
	if *noPodman {
		v := false
		podmanVal = &v
	} else if *podman {
		v := true
		podmanVal = &v
	}
	return InitFlags{
		Podman: podmanVal,
	}, nil
}
