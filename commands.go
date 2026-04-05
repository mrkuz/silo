package main

import (
	"flag"
	"fmt"
	"strings"
)

// cmdInit implements `silo init`.
func cmdInit() error {
	_, err := ensureInit()
	if err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}
	return nil
}

// cmdSetup runs post-start setup in the running container.
func cmdSetup() error {
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	if !containerRunning(cfg.General.ContainerName) {
		return fmt.Errorf("container %s is not running", cfg.General.ContainerName)
	}
	if err := setupContainer(cfg); err != nil {
		return fmt.Errorf("setup container: %w", err)
	}
	return nil
}

// cmdConnect opens an interactive shell in the running container.
func cmdConnect(args []string) error {
	flags, err := parseConnectFlags(args)
	if err != nil {
		return fmt.Errorf("parse connect flags: %w", err)
	}
	cfg, err := ensureSetup()
	if err != nil {
		return fmt.Errorf("setup container: %w", err)
	}
	fmt.Printf("Connecting to %s...\n", cfg.General.ContainerName)
	err = connectContainer(cfg.General.ContainerName, cfg.Connect.Command, flags.extra)
	if flags.stop {
		// Best-effort cleanup; original session error (if any) takes precedence.
		stopContainer(cfg.General.ContainerName)
	}
	if err != nil {
		return fmt.Errorf("connect to container: %w", err)
	}
	return nil
}

// cmdExec runs a command in the running container.
func cmdExec(args []string) error {
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	if !containerRunning(cfg.General.ContainerName) {
		return fmt.Errorf("container %s is not running", cfg.General.ContainerName)
	}
	execArgs := append([]string{"exec", "-ti", cfg.General.ContainerName}, args...)
	if err := runInteractive("podman", execArgs...); err != nil {
		return fmt.Errorf("execute command in container: %w", err)
	}
	return nil
}

// cmdStop implements `silo stop`.
func cmdStop() error {
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	if !containerRunning(cfg.General.ContainerName) {
		fmt.Printf("%s is not running\n", cfg.General.ContainerName)
		return nil
	}
	if err := stopContainer(cfg.General.ContainerName); err != nil {
		return fmt.Errorf("stop container: %w", err)
	}
	return nil
}

// cmdStatus implements `silo status`.
func cmdStatus() error {
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	if containerRunning(cfg.General.ContainerName) {
		fmt.Println("Running")
	} else {
		fmt.Println("Stopped")
	}
	return nil
}

// cmdRemove removes the workspace container and optionally the image.
func cmdRemove(args []string) error {
	flags, err := parseRemoveFlags(args)
	if err != nil {
		return fmt.Errorf("parse remove flags: %w", err)
	}
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	if containerExists(cfg.General.ContainerName) {
		if containerRunning(cfg.General.ContainerName) {
			if !flags.force {
				return fmt.Errorf("%s is running", cfg.General.ContainerName)
			}
			if err := stopContainer(cfg.General.ContainerName); err != nil {
				return fmt.Errorf("stop container before removal: %w", err)
			}
		}
		fmt.Printf("Removing %s...\n", cfg.General.ContainerName)
		if err := removeContainer(cfg.General.ContainerName); err != nil {
			return fmt.Errorf("remove container: %w", err)
		}
	} else {
		fmt.Printf("%s not found\n", cfg.General.ContainerName)
	}
	if flags.image {
		if imageExists(cfg.General.ImageName) {
			fmt.Printf("Removing %s...\n", cfg.General.ImageName)
			if err := removeImage(cfg.General.ImageName); err != nil {
				return fmt.Errorf("remove image: %w", err)
			}
		} else {
			fmt.Printf("%s not found\n", cfg.General.ImageName)
		}
	}
	return nil
}

// cmdCreate creates the container from the workspace image.
func cmdCreate(args []string) error {
	flags, err := parseCreateFlags(args)
	if err != nil {
		return fmt.Errorf("parse create flags: %w", err)
	}
	cfg, err := ensureBuilt()
	if err != nil {
		return fmt.Errorf("build images: %w", err)
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
		podmanArgs, err := buildContainerArgs(cfg)
		if err != nil {
			return fmt.Errorf("build container arguments: %w", err)
		}
		createArgs := append([]string{"create"}, podmanArgs...)
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
		fmt.Printf("Removing %s...\n", cfg.General.ContainerName)
		if err := removeContainer(cfg.General.ContainerName); err != nil {
			return fmt.Errorf("remove existing container: %w", err)
		}
	}

	if err := createContainer(cfg, extraArgs); err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	return nil
}

// cmdStart implements `silo start [--force]`.
func cmdStart(args []string) error {
	flags, err := parseStartFlags(args)
	if err != nil {
		return fmt.Errorf("parse start flags: %w", err)
	}
	if flags.force {
		// Stop first if running, then let the ensure chain handle start + setup.
		cfg, err := ensureBuilt()
		if err != nil {
			return fmt.Errorf("build images: %w", err)
		}
		if containerRunning(cfg.General.ContainerName) {
			if err := stopContainer(cfg.General.ContainerName); err != nil {
				return fmt.Errorf("stop container: %w", err)
			}
		}
	}
	_, err = ensureSetup()
	if err != nil {
		return fmt.Errorf("setup container: %w", err)
	}
	return nil
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
		return connectFlags{}, fmt.Errorf("parse connect flags: %w", err)
	}
	return connectFlags{stop: *stop, extra: fs.Args()}, nil
}

type removeFlags struct {
	force bool
	image bool
}

func parseRemoveFlags(args []string) (removeFlags, error) {
	fs := flag.NewFlagSet("silo rm", flag.ContinueOnError)
	force := fs.Bool("force", false, "Stop and remove a running container")
	f := fs.Bool("f", false, "Alias for --force")
	image := fs.Bool("image", false, "Also remove the workspace image")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return removeFlags{}, fmt.Errorf("parse remove flags: %w", err)
	}
	return removeFlags{force: *force || *f, image: *image}, nil
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
	f := fs.Bool("f", false, "Alias for --force")
	dryRun := fs.Bool("dry-run", false, "Print the podman create command without running it")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return createFlags{}, fmt.Errorf("parse create flags: %w", err)
	}
	return createFlags{
		nested:         *nested,
		noWorkspace:    *noWorkspace,
		noSharedVolume: *noSharedVolume,
		force:          *force || *f,
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
	f := fs.Bool("f", false, "Alias for --force")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return startFlags{}, fmt.Errorf("parse start flags: %w", err)
	}
	return startFlags{force: *force || *f}, nil
}
