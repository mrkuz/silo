package cmd

import (
	"flag"
	"fmt"

	"github.com/mrkuz/silo/internal"
)

// Run implements default `silo` invocation.
func Run(args []string) error {
	flags, err := ParseRunFlags(args)
	if err != nil {
		return fmt.Errorf("parse run flags: %w", err)
	}
	cfg, err := internal.EnsureStarted()
	if err != nil {
		return fmt.Errorf("setup container: %w", err)
	}
	fmt.Printf("Connecting to %s...\n", cfg.General.ContainerName)
	err = internal.ConnectContainer(cfg.General.ContainerName, cfg.Connect.Command, flags.Extra)
	// Best-effort cleanup; original session error (if any) takes precedence.
	if flags.Stop {
		internal.StopContainer(cfg.General.ContainerName)
		internal.RemoveContainer(cfg.General.ContainerName)
	}
	if flags.Remove {
		internal.StopContainer(cfg.General.ContainerName)
		internal.RemoveContainer(cfg.General.ContainerName)
		internal.RemoveImage(cfg.General.ImageName)
	}
	if err != nil {
		return fmt.Errorf("connect to container: %w", err)
	}
	return nil
}

// Connect opens an interactive shell in the running container.
func Connect(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("connect does not take arguments")
	}
	cfg, err := internal.RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	if !internal.ContainerExists(cfg.General.ContainerName) {
		return fmt.Errorf("container %s does not exist", cfg.General.ContainerName)
	}
	if !internal.ContainerRunning(cfg.General.ContainerName) {
		return fmt.Errorf("container %s is not running", cfg.General.ContainerName)
	}
	fmt.Printf("Connecting to %s...\n", cfg.General.ContainerName)
	if err := internal.ConnectContainer(cfg.General.ContainerName, cfg.Connect.Command, nil); err != nil {
		return fmt.Errorf("connect to container: %w", err)
	}
	return nil
}

// RunFlags holds parsed flags for the run command.
type RunFlags struct {
	Stop    bool
	Remove  bool
	Extra   []string
}

// ParseRunFlags parses the flags for the default run command.
func ParseRunFlags(args []string) (RunFlags, error) {
	fs := flag.NewFlagSet("silo", flag.ContinueOnError)
	stop := fs.Bool("stop", false, "Stop and remove the container when the session exits")
	rm := fs.Bool("rm", false, "Stop, remove container, and remove image when the session exits")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return RunFlags{}, fmt.Errorf("parse run flags: %w", err)
	}
	return RunFlags{Stop: *stop || *rm, Remove: *rm, Extra: fs.Args()}, nil
}
