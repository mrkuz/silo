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

// cmdRun implements default `silo` invocation.
func cmdRun(args []string) error {
	flags, err := parseRunFlags(args)
	if err != nil {
		return fmt.Errorf("parse run flags: %w", err)
	}
	cfg, err := ensureSetup()
	if err != nil {
		return fmt.Errorf("setup container: %w", err)
	}
	fmt.Printf("Connecting to %s...\n", cfg.General.ContainerName)
	err = connectContainer(cfg.General.ContainerName, cfg.Connect.Command, flags.extra)
	// Best-effort cleanup; original session error (if any) takes precedence.
	if flags.stop {
		stopContainer(cfg.General.ContainerName)
	}
	if flags.rm {
		removeContainer(cfg.General.ContainerName)
	}
	if flags.rmi {
		removeImage(cfg.General.ImageName)
	}
	if err != nil {
		return fmt.Errorf("connect to container: %w", err)
	}
	return nil
}

// cmdConnect opens an interactive shell in the running container.
func cmdConnect(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("connect does not take arguments")
	}
	cfg, err := ensureSetup()
	if err != nil {
		return fmt.Errorf("setup container: %w", err)
	}
	fmt.Printf("Connecting to %s...\n", cfg.General.ContainerName)
	if err := connectContainer(cfg.General.ContainerName, cfg.Connect.Command, nil); err != nil {
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

// cmdRemove removes the workspace container.
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
	return nil
}

// cmdRemoveImage implements `silo rmi [--force] [--user]`.
func cmdRemoveImage(args []string) error {
	flags, err := parseRemoveImageFlags(args)
	if err != nil {
		return fmt.Errorf("parse rmi flags: %w", err)
	}
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	if flags.user {
		userImage := baseImageName(cfg.General.User)
		if imageExists(userImage) {
			fmt.Printf("Removing %s...\n", userImage)
			if err := removeImage(userImage); err != nil {
				return fmt.Errorf("remove user image: %w", err)
			}
		} else {
			fmt.Printf("%s not found\n", userImage)
		}
		return nil
	}
	if flags.force {
		if containerExists(cfg.General.ContainerName) {
			if containerRunning(cfg.General.ContainerName) {
				if err := stopContainer(cfg.General.ContainerName); err != nil {
					return fmt.Errorf("stop container: %w", err)
				}
			}
			fmt.Printf("Removing %s...\n", cfg.General.ContainerName)
			if err := removeContainer(cfg.General.ContainerName); err != nil {
				return fmt.Errorf("remove container: %w", err)
			}
		}
	}
	if imageExists(cfg.General.ImageName) {
		fmt.Printf("Removing %s...\n", cfg.General.ImageName)
		if err := removeImage(cfg.General.ImageName); err != nil {
			return fmt.Errorf("remove image: %w", err)
		}
	} else {
		fmt.Printf("%s not found\n", cfg.General.ImageName)
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
	if flags.sharedVolume && !cfg.Features.SharedVolume {
		cfg.Features.SharedVolume = true
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
		fmt.Printf("%s already exists\n", cfg.General.ContainerName)
		return nil
	}

	if err := createContainer(cfg, extraArgs); err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	return nil
}

// cmdStart implements `silo start`.
func cmdStart() error {
	cfg, err := ensureCreated()
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	if containerRunning(cfg.General.ContainerName) {
		fmt.Printf("%s is already running\n", cfg.General.ContainerName)
		return nil
	}
	if err := startContainer(cfg.General.ContainerName); err != nil {
		return fmt.Errorf("start container: %w", err)
	}
	err = setupContainer(cfg)
	if err != nil {
		return fmt.Errorf("setup container: %w", err)
	}
	return nil
}

// --- flag types and parsers ---

type runFlags struct {
	stop  bool
	rm    bool
	rmi   bool
	extra []string
}

func parseRunFlags(args []string) (runFlags, error) {
	fs := flag.NewFlagSet("silo", flag.ContinueOnError)
	stop := fs.Bool("stop", false, "Stop the container when the session exits")
	rm := fs.Bool("rm", false, "Stop and remove the container when the session exits")
	rmi := fs.Bool("rmi", false, "Stop, remove container, and remove image when the session exits")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return runFlags{}, fmt.Errorf("parse run flags: %w", err)
	}
	return runFlags{stop: *stop || *rm || *rmi, rm: *rm || *rmi, rmi: *rmi, extra: fs.Args()}, nil
}

type removeFlags struct {
	force bool
}

func parseRemoveFlags(args []string) (removeFlags, error) {
	fs := flag.NewFlagSet("silo rm", flag.ContinueOnError)
	force := fs.Bool("force", false, "Stop and remove a running container")
	forceShort := fs.Bool("f", false, "")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return removeFlags{}, fmt.Errorf("parse remove flags: %w", err)
	}
	return removeFlags{force: *force || *forceShort}, nil
}

type removeImageFlags struct {
	force bool
	user  bool
}

func parseRemoveImageFlags(args []string) (removeImageFlags, error) {
	fs := flag.NewFlagSet("silo rmi", flag.ContinueOnError)
	force := fs.Bool("force", false, "Stop and remove the container before removing the image")
	forceShort := fs.Bool("f", false, "")
	user := fs.Bool("user", false, "Remove only the user image")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return removeImageFlags{}, fmt.Errorf("parse rmi flags: %w", err)
	}
	return removeImageFlags{force: *force || *forceShort, user: *user}, nil
}

type devcontainerRemoveFlags struct {
	force bool
}

func parseDevcontainerRemoveFlags(args []string) (devcontainerRemoveFlags, error) {
	fs := flag.NewFlagSet("silo devcontainer rm", flag.ContinueOnError)
	force := fs.Bool("force", false, "Stop and remove a running container")
	forceShort := fs.Bool("f", false, "")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return devcontainerRemoveFlags{}, fmt.Errorf("parse devcontainer rm flags: %w", err)
	}
	return devcontainerRemoveFlags{force: *force || *forceShort}, nil
}

type createFlags struct {
	nested       bool
	sharedVolume bool
	dryRun       bool
	extra        []string
}

func parseCreateFlags(args []string) (createFlags, error) {
	fs := flag.NewFlagSet("silo create", flag.ContinueOnError)
	nested := fs.Bool("nested", false, "Enable nested Podman containers")
	sharedVolume := fs.Bool("shared-volume", false, "Enable shared volume")
	dryRun := fs.Bool("dry-run", false, "Print the podman create command without running it")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return createFlags{}, fmt.Errorf("parse create flags: %w", err)
	}
	return createFlags{
		nested:       *nested,
		sharedVolume: *sharedVolume,
		dryRun:       *dryRun,
		extra:        fs.Args(),
	}, nil
}
