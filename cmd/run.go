package cmd

import (
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Run implements default `silo` invocation.
func Run(args []string) error {
	flags, err := ParseRunFlags(args)
	if err != nil {
		return err
	}
	cfg, err := internal.EnsureStarted()
	if err != nil {
		return fmt.Errorf("setup container: %w", err)
	}
	containerName := internal.WorkspaceContainerName(cfg.General.ID)
	fmt.Printf("Connecting to %s...\n", containerName)
	err = internal.ConnectContainer(containerName)
	// Best-effort cleanup; original session error (if any) takes precedence.
	if flags.Stop {
		internal.StopContainer(containerName)
		internal.RemoveContainer(containerName)
	}
	if err != nil {
		return fmt.Errorf("connect to container: %w", err)
	}
	return nil
}

// RunFlags holds parsed flags for the run command.
type RunFlags struct {
	Stop bool
}

// ParseRunFlags parses the flags for the default run command.
func ParseRunFlags(args []string) (RunFlags, error) {
	fs := NewFlagSet("silo")
	stop := fs.Bool("stop", false, "Stop and remove the container when the session exits")
	if err := parseWithInterceptor(fs, args); err != nil {
		return RunFlags{}, err
	}
	if len(fs.Args()) > 0 {
		return RunFlags{}, ErroneousCommand()
	}
	return RunFlags{Stop: *stop}, nil
}
