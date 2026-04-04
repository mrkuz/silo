package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var execCommand = exec.Command

// containerRunning reports whether the named container is currently running.
func containerRunning(name string) bool {
	out, err := execCommand("podman", "container", "inspect", "--format", "{{.State.Running}}", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// containerExists reports whether a container with the given name exists (any state).
func containerExists(name string) bool {
	return execCommand("podman", "container", "exists", name).Run() == nil
}

// printDryRun prints a podman command as it would be invoked, without running it.
func printDryRun(args []string) {
	fmt.Println("podman " + strings.Join(args, " "))
}

// connectContainer attaches to a running container via podman exec.
// Extra args are inserted before the container name as podman exec flags.
func connectContainer(name, command string, extra []string) error {
	args := append([]string{"exec", "-ti"}, extra...)
	args = append(args, name)
	args = append(args, strings.Fields(command)...)
	return runInteractive("podman", args...)
}

// securityArgs returns the podman security-related flags for the given nesting mode.
func securityArgs(nested bool) []string {
	if nested {
		return []string{"--security-opt", "label=disable", "--device", "/dev/fuse"}
	}
	return []string{"--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges"}
}

// buildContainerArgs constructs the podman container-specific arguments from cfg,
// without any subcommand prefix. Callers prepend "run", "-d" or "create" as needed.
func buildContainerArgs(cfg Config) ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	var args []string

	// Security options
	args = append(args, securityArgs(cfg.Features.Nested)...)

	args = append(args, "--name", cfg.General.ContainerName)
	args = append(args, "--hostname", cfg.General.ContainerName)
	args = append(args, "--user", cfg.General.User)

	// Workspace mount
	if cfg.Features.Workspace {
		dirName := filepath.Base(cwd)
		mountPath := fmt.Sprintf("/workspace/%s/%s", cfg.General.ID, dirName)
		args = append(args, "--volume", fmt.Sprintf("%s:%s:Z", cwd, mountPath))
		args = append(args, "--workdir", mountPath)
	}

	// Shared volume
	if cfg.Features.SharedVolume {
		args = append(args, "--volume", sharedVolume+":/shared:Z")
	}

	return args, nil
}

// sharedVolumeEntry holds pre-computed source and destination paths for a shared volume symlink.
type sharedVolumeEntry struct {
	Src   string
	Dst   string
	IsDir bool
}

// buildSharedVolumeEntries converts raw path strings into pre-computed entries.
// Paths ending in '/' are directories; others are files.
// $HOME prefixes are expanded to ${HOME} for shell runtime expansion.
func buildSharedVolumeEntries(paths []string) []sharedVolumeEntry {
	entries := make([]sharedVolumeEntry, 0, len(paths))
	for _, raw := range paths {
		isDir := len(raw) > 0 && raw[len(raw)-1] == '/'
		dst := strings.TrimRight(raw, "/")
		var src string
		if strings.HasPrefix(dst, "$HOME") {
			src = "/shared${HOME}" + dst[len("$HOME"):]
		} else {
			src = "/shared" + dst
		}
		entries = append(entries, sharedVolumeEntry{Src: src, Dst: dst, IsDir: isDir})
	}
	return entries
}

// setupContainer runs post-start setup inside a running container.
// Currently this renders and pipes the shared volume setup script via stdin.
func setupContainer(cfg Config) error {
	if !cfg.Features.SharedVolume || len(cfg.SharedVolume.Paths) == 0 {
		return nil
	}
	entries := buildSharedVolumeEntries(cfg.SharedVolume.Paths)
	script, err := renderTemplate("setup.sh.tmpl", struct{ Entries []sharedVolumeEntry }{entries})
	if err != nil {
		return fmt.Errorf("render setup script: %w", err)
	}
	cmd := execCommand("podman", "exec", "-i", cfg.General.ContainerName, "bash")
	cmd.Stdin = bytes.NewReader(script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("container setup: %w", err)
	}
	return nil
}

// createContainer creates and starts a new container, running shared volume setup if needed.
// The container is left running. extra args are forwarded to podman create.
func createContainer(cfg Config, extra []string) error {
	containerArgs, err := buildContainerArgs(cfg)
	if err != nil {
		return err
	}
	createArgs := append([]string{"create"}, containerArgs...)
	createArgs = append(createArgs, extra...)
	createArgs = append(createArgs, cfg.General.ImageName)

	fmt.Printf("Creating %s...\n", cfg.General.ContainerName)
	cmd := execCommand("podman", createArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	if err := execCommand("podman", "start", cfg.General.ContainerName).Run(); err != nil {
		return err
	}

	return setupContainer(cfg)
}

// ensureContainerRunning ensures the container exists and is running,
// creating it if needed or starting it if stopped.
func ensureContainerRunning(cfg Config) error {
	if !containerExists(cfg.General.ContainerName) {
		return createContainer(cfg, cfg.Create.ExtraArgs)
	}
	if !containerRunning(cfg.General.ContainerName) {
		fmt.Printf("Starting %s...\n", cfg.General.ContainerName)
		if err := execCommand("podman", "start", cfg.General.ContainerName).Run(); err != nil {
			return err
		}
		return setupContainer(cfg)
	}
	return nil
}

// removeContainer forcibly removes the named container.
func removeContainer(name string) error {
	cmd := execCommand("podman", "rm", "-f", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// removeImage removes the named image.
func removeImage(name string) error {
	cmd := execCommand("podman", "rmi", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runInteractive runs a command with stdio connected to the terminal.
func runInteractive(name string, args ...string) error {
	cmd := execCommand(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// cmdInit implements `silo init`.
// Creates global scaffold files and local .silo/silo.toml + .silo/home.nix.
func cmdInit() error {
	if _, err := initWorkspaceConfig(); err != nil {
		return err
	}
	return ensureScaffoldFiles()
}

// cmdSetup implements `silo setup`.
// Runs the post-start setup script inside the running container.
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

	cfg, err := initWorkspaceConfig()
	if err != nil {
		return err
	}

	if err := ensureScaffoldFiles(); err != nil {
		return err
	}
	if err := ensureImages(cfg); err != nil {
		return err
	}

	if err := ensureContainerRunning(cfg); err != nil {
		return err
	}
	fmt.Printf("Connecting to %s...\n", cfg.General.ContainerName)
	err = connectContainer(cfg.General.ContainerName, cfg.Connect.Command, flags.extra)
	if flags.stop {
		// Ignore error — best-effort cleanup; original session error (if any) takes precedence.
		execCommand("podman", "stop", "-t", "0", cfg.General.ContainerName).Run()
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
	fmt.Printf("Stopping %s...\n", cfg.General.ContainerName)
	return execCommand("podman", "stop", "-t", "0", cfg.General.ContainerName).Run()
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
			fmt.Printf("Stopping %s...\n", cfg.General.ContainerName)
			if err := execCommand("podman", "stop", "-t", "0", cfg.General.ContainerName).Run(); err != nil {
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

	cfg, err := initWorkspaceConfig()
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

	if err := ensureScaffoldFiles(); err != nil {
		return err
	}
	if err := ensureImages(cfg); err != nil {
		return err
	}

	if containerExists(cfg.General.ContainerName) {
		if !flags.force {
			fmt.Printf("Container %s already exists.\n", cfg.General.ContainerName)
			return nil
		}
		fmt.Printf("Removing existing container %s...\n", cfg.General.ContainerName)
		if err := removeContainer(cfg.General.ContainerName); err != nil {
			return err
		}
	}

	if err := createContainer(cfg, extraArgs); err != nil {
		return err
	}
	return execCommand("podman", "stop", "-t", "0", cfg.General.ContainerName).Run()
}

// cmdStart implements `silo start [--force]`.
func cmdStart(args []string) error {
	flags, err := parseStartFlags(args)
	if err != nil {
		return err
	}

	cfg, err := initWorkspaceConfig()
	if err != nil {
		return err
	}

	if err := ensureScaffoldFiles(); err != nil {
		return err
	}
	if err := ensureImages(cfg); err != nil {
		return err
	}

	// If already running, stop first (force) or bail.
	if containerRunning(cfg.General.ContainerName) {
		if !flags.force {
			fmt.Printf("Container %s is already running.\n", cfg.General.ContainerName)
			return nil
		}
		fmt.Printf("Stopping %s...\n", cfg.General.ContainerName)
		if err := execCommand("podman", "stop", "-t", "0", cfg.General.ContainerName).Run(); err != nil {
			return err
		}
	}

	// Create the container if it doesn't exist yet.
	if !containerExists(cfg.General.ContainerName) {
		return createContainer(cfg, nil)
	}

	fmt.Printf("Starting %s...\n", cfg.General.ContainerName)
	if err := execCommand("podman", "start", cfg.General.ContainerName).Run(); err != nil {
		return err
	}
	return setupContainer(cfg)
}

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
