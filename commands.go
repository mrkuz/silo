package main

import (
	"flag"
	"fmt"
	"strings"
)

// cmdInit implements `silo init`.
func cmdInit() error {
	_, err := ensureInit()
	return err
}

// cmdSetup implements `silo setup`.
func cmdSetup() error {
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return err
	}
	if !containerRunning(cfg.General.ContainerName) {
		return fmt.Errorf("container %s is not running", cfg.General.ContainerName)
	}
	return setupContainer(cfg)
}

// cmdConnect implements `silo connect [--stop] [-- extra...]` and is the default command.
func cmdConnect(args []string) error {
	flags, err := parseConnectFlags(args)
	if err != nil {
		return err
	}
	cfg, err := ensureSetup()
	if err != nil {
		return err
	}
	fmt.Printf("Connecting to %s...\n", cfg.General.ContainerName)
	err = connectContainer(cfg.General.ContainerName, cfg.Connect.Command, flags.extra)
	if flags.stop {
		// Best-effort cleanup; original session error (if any) takes precedence.
		stopContainer(cfg.General.ContainerName)
	}
	return err
}

// cmdExec implements `silo exec <cmd> [args...]`.
func cmdExec(args []string) error {
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return err
	}
	if !containerRunning(cfg.General.ContainerName) {
		return fmt.Errorf("container %s is not running", cfg.General.ContainerName)
	}
	execArgs := append([]string{"exec", "-ti", cfg.General.ContainerName}, args...)
	return runInteractive("podman", execArgs...)
}

// cmdStop implements `silo stop`.
func cmdStop() error {
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return err
	}
	if !containerRunning(cfg.General.ContainerName) {
		fmt.Printf("Container %s is not running.\n", cfg.General.ContainerName)
		return nil
	}
	return stopContainer(cfg.General.ContainerName)
}

// cmdStatus implements `silo status`.
func cmdStatus() error {
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return err
	}
	if containerRunning(cfg.General.ContainerName) {
		fmt.Println("Running")
	} else {
		fmt.Println("Stopped")
	}
	return nil
}

// cmdRemove implements `silo rm [--image]`.
func cmdRemove(args []string) error {
	removeImg, err := parseRemoveFlags(args)
	if err != nil {
		return err
	}
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return err
	}
	if containerExists(cfg.General.ContainerName) {
		if containerRunning(cfg.General.ContainerName) {
			if err := stopContainer(cfg.General.ContainerName); err != nil {
				return err
			}
		}
		fmt.Printf("Removing container %s...\n", cfg.General.ContainerName)
		if err := removeContainer(cfg.General.ContainerName); err != nil {
			return err
		}
	} else {
		fmt.Printf("No container %s found.\n", cfg.General.ContainerName)
	}
	if removeImg {
		if imageExists(cfg.General.ImageName) {
			fmt.Printf("Removing image %s...\n", cfg.General.ImageName)
			return removeImage(cfg.General.ImageName)
		}
		fmt.Printf("No image %s found.\n", cfg.General.ImageName)
	}
	return nil
}

// cmdCreate implements `silo create [--nested] [--no-workspace] [--no-shared-volume] [--force] [--dry-run] [-- extra...]`.
func cmdCreate(args []string) error {
	flags, err := parseCreateFlags(args)
	if err != nil {
		return err
	}
	cfg, err := ensureBuilt()
	if err != nil {
		return err
	}

	// Apply feature flags to cfg; persist to silo.toml only when not a dry run.
	changed := false
	if flags.nested && !cfg.Features.Nested {
		cfg.Features.Nested = true
		changed = true
	}
	if flags.noWorkspace && cfg.Features.Workspace {
		cfg.Features.Workspace = false
		changed = true
	}
	if flags.noSharedVolume && cfg.Features.SharedVolume {
		cfg.Features.SharedVolume = false
		changed = true
	}

	// Resolve extra args: CLI wins over stored value; update stored value when CLI provides args.
	var extraArgs []string
	if len(flags.extra) > 0 {
		extraArgs = flags.extra
		if strings.Join(flags.extra, "\x00") != strings.Join(cfg.Create.ExtraArgs, "\x00") {
			cfg.Create.ExtraArgs = flags.extra
			changed = true
		}
	} else {
		extraArgs = cfg.Create.ExtraArgs
	}

	if flags.dryRun {
		containerArgs, err := buildContainerArgs(cfg)
		if err != nil {
			return err
		}
		createArgs := append([]string{"create"}, containerArgs...)
		createArgs = append(createArgs, extraArgs...)
		createArgs = append(createArgs, cfg.General.ImageName)
		printDryRun(createArgs)
		return nil
	}

	if changed {
		if err := cfg.saveWorkspaceConfig(); err != nil {
			return fmt.Errorf("save silo.toml: %w", err)
		}
	}

	if containerExists(cfg.General.ContainerName) {
		if !flags.force {
			fmt.Printf("Container %s already exists.\n", cfg.General.ContainerName)
			return nil
		}
		fmt.Printf("Removing container %s...\n", cfg.General.ContainerName)
		if err := removeContainer(cfg.General.ContainerName); err != nil {
			return err
		}
	}

	return createContainer(cfg, extraArgs)
}

// cmdStart implements `silo start [--force]`.
func cmdStart(args []string) error {
	flags, err := parseStartFlags(args)
	if err != nil {
		return err
	}
	if flags.force {
		// Stop first if running, then let the ensure chain handle start + setup.
		cfg, err := ensureBuilt()
		if err != nil {
			return err
		}
		if containerRunning(cfg.General.ContainerName) {
			if err := stopContainer(cfg.General.ContainerName); err != nil {
				return err
			}
		}
	}
	_, err = ensureSetup()
	return err
}

// --- flag types and parsers ---

type connectFlags struct {
	stop  bool
	extra []string
}

func parseConnectFlags(args []string) (connectFlags, error) {
	fs := flag.NewFlagSet("silo connect", flag.ContinueOnError)
	stop := fs.Bool("stop", false, "Stop the container when the session exits")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return connectFlags{}, err
	}
	return connectFlags{stop: *stop, extra: fs.Args()}, nil
}

func parseRemoveFlags(args []string) (bool, error) {
	fs := flag.NewFlagSet("silo rm", flag.ContinueOnError)
	removeImg := fs.Bool("image", false, "Also remove the workspace image")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return false, err
	}
	return *removeImg, nil
}

type createFlags struct {
	nested         bool
	noWorkspace    bool
	noSharedVolume bool
	force          bool
	dryRun         bool
	extra          []string
}

func parseCreateFlags(args []string) (createFlags, error) {
	fs := flag.NewFlagSet("silo create", flag.ContinueOnError)
	nested := fs.Bool("nested", false, "Enable nested Podman containers")
	noWorkspace := fs.Bool("no-workspace", false, "Disable workspace volume mount")
	noSharedVolume := fs.Bool("no-shared-volume", false, "Disable shared volume")
	force := fs.Bool("force", false, "Remove and recreate the container if it already exists")
	dryRun := fs.Bool("dry-run", false, "Print the podman create command without running it")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return createFlags{}, err
	}
	return createFlags{
		nested:         *nested,
		noWorkspace:    *noWorkspace,
		noSharedVolume: *noSharedVolume,
		force:          *force,
		dryRun:         *dryRun,
		extra:          fs.Args(),
	}, nil
}

type startFlags struct {
	force bool
}

func parseStartFlags(args []string) (startFlags, error) {
	fs := flag.NewFlagSet("silo start", flag.ContinueOnError)
	force := fs.Bool("force", false, "Restart the container if it is already running")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return startFlags{}, err
	}
	return startFlags{force: *force}, nil
}
