package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// cmdInit implements `silo init`. Creates workspace files only.
// Use `silo user init` to create user files.
func cmdInit() error {
	var err error
	initPaths := []string{siloToml, filepath.Join(siloDir, "home.nix")}
	for _, p := range initPaths {
		if err := printInitFileStatus(p); err != nil {
			return err
		}
	}

	_, err = ensureInit()
	if err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}
	return nil
}

// cmdUserInit implements `silo user init`. Prints per-file status
// (for existing and new files) and delegates the actual
// file creation to ensureUserFiles.
func cmdUserInit() error {
	files, err := userStarterFiles()
	if err != nil {
		return fmt.Errorf("list user starter files: %w", err)
	}
	for _, f := range files {
		if err := printInitFileStatus(f.path); err != nil {
			return err
		}
	}
	if err := ensureUserFiles(); err != nil {
		return fmt.Errorf("create user files: %w", err)
	}
	return nil
}

func printInitFileStatus(path string) error {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("'%s' already exists\n", path)
		return nil
	} else if os.IsNotExist(err) {
		fmt.Printf("Creating %s\n", path)
		return nil
	} else {
		return fmt.Errorf("stat %s: %w", path, err)
	}
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
	printRunningStatus(containerRunning(cfg.General.ContainerName))
	return nil
}

func printRunningStatus(isRunning bool) {
	if isRunning {
		fmt.Println("Running")
		return
	}
	fmt.Println("Stopped")
}

func removeNamedContainer(name string, force bool) error {
	if !containerExists(name) {
		fmt.Printf("%s not found\n", name)
		return nil
	}
	if containerRunning(name) {
		if !force {
			return fmt.Errorf("%s is running", name)
		}
		if err := stopContainer(name); err != nil {
			return fmt.Errorf("stop container before removal: %w", err)
		}
	}
	fmt.Printf("Removing %s...\n", name)
	if err := removeContainer(name); err != nil {
		return fmt.Errorf("remove container: %w", err)
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
	if err := removeNamedContainer(cfg.General.ContainerName, flags.force); err != nil {
		return err
	}
	return nil
}

// cmdRemoveImage implements `silo rmi [--force]`.
func cmdRemoveImage(args []string) error {
	flags, err := parseRemoveImageFlags(args)
	if err != nil {
		return fmt.Errorf("parse rmi flags: %w", err)
	}
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
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
	force, err := parseForceFlag(args, "silo rm", "Stop and remove a running container", "parse remove flags")
	if err != nil {
		return removeFlags{}, err
	}
	return removeFlags{force: force}, nil
}

type removeImageFlags struct {
	force bool
}

func parseRemoveImageFlags(args []string) (removeImageFlags, error) {
	force, err := parseForceFlag(args, "silo rmi", "Stop and remove the container before removing the image", "parse rmi flags")
	if err != nil {
		return removeImageFlags{}, err
	}
	return removeImageFlags{force: force}, nil
}

// cmdUserRmi implements `silo user rmi`. Removes the user image.
func cmdUserRmi() error {
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
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

type devcontainerRemoveFlags struct {
	force bool
}

func parseDevcontainerRemoveFlags(args []string) (devcontainerRemoveFlags, error) {
	force, err := parseForceFlag(args, "silo devcontainer rm", "Stop and remove a running container", "parse devcontainer rm flags")
	if err != nil {
		return devcontainerRemoveFlags{}, err
	}
	return devcontainerRemoveFlags{force: force}, nil
}

func parseForceFlag(args []string, name, usage, context string) (bool, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	force := fs.Bool("force", false, usage)
	forceShort := fs.Bool("f", false, "")
	fs.Usage = func() {} // suppress; handled by main helpText
	if err := fs.Parse(args); err != nil {
		return false, fmt.Errorf("%s: %w", context, err)
	}
	return *force || *forceShort, nil
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
