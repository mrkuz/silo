package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var execCommand = exec.Command

// resolveContainerPath converts a shared volume path to its container mount target.
// - Absolute paths (e.g., "/etc/shared/") are used directly.
// - $HOME prefix (e.g., "$HOME/.cache/uv/") expands to "/home/<user>/.cache/uv".
// - Relative paths are not supported - a warning is printed and the path is skipped.
func resolveContainerPath(path string, user string) string {
	const homePrefix = "$HOME"
	// Normalize path: remove trailing slashes
	path = strings.TrimRight(path, "/")

	// Absolute path: use directly
	if strings.HasPrefix(path, "/") {
		return path
	}

	// $HOME prefix: expand to container home path
	if strings.HasPrefix(path, homePrefix) {
		rest := path[len(homePrefix):]
		if rest == "" {
			return "/home/" + user
		}
		return "/home/" + user + rest
	}

	// Relative paths are not supported - return empty string
	fmt.Fprintf(os.Stderr, "warning: relative paths are not supported in shared volume config: %s\n", path)
	return ""
}

// volumeSetup creates directories on the silo-shared volume from the host side
// by running a temporary container with the user image, ensuring directories exist
// before they are mounted as subpath volumes.
func volumeSetup(cfg Config) error {
	if !cfg.Features.SharedVolume || len(cfg.SharedVolume.Paths) == 0 {
		return nil
	}

	// Ensure the user image exists before using it for volume setup
	userImage := baseImageName(cfg.General.User)
	if !imageExists(userImage) {
		tc, err := newTemplateContext(cfg)
		if err != nil {
			return fmt.Errorf("build template context: %w", err)
		}
		if err := ensureUserImage(tc); err != nil {
			return fmt.Errorf("ensure user image: %w", err)
		}
	}

	var mkdirCmd strings.Builder
	for i, path := range cfg.SharedVolume.Paths {
		containerPath := resolveContainerPath(path, cfg.General.User)
		// Skip invalid/relative paths (resolveContainerPath prints a warning)
		if containerPath == "" {
			continue
		}
		if i > 0 {
			mkdirCmd.WriteString(" && ")
		}
		// Check if it's a directory (original path ends with /) or file
		isDir := len(path) > 0 && path[len(path)-1] == '/'
		// The volume is mounted at /silo/shared, so prefix paths accordingly
		volPath := volumeMountPath + containerPath
		if isDir {
			// Directory: create with chmod 755
			mkdirCmd.WriteString("mkdir -p " + volPath + " && chmod 755 " + volPath)
		} else {
			// File: create parent dir, touch file, set permissions
			mkdirCmd.WriteString("mkdir -p $(dirname " + volPath + ") && touch " + volPath + " && chmod 644 " + volPath)
		}
	}
	// Use the user image to create directories inside the volume.
	// Mount the volume at /silo/shared so we can create subdirectories in it.
	cmd := execCommand("podman", "run", "--rm", "-v", cfg.getSharedVolumeName()+":"+volumeMountPath+":Z", userImage, "sh", "-c", mkdirCmd.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("volume setup: %w", err)
	}
	return nil
}

// containerRunning checks if a container is currently running.
func containerRunning(name string) bool {
	out, err := execCommand("podman", "container", "inspect", "--format", "{{.State.Running}}", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// containerExists checks if a container exists in any state.
func containerExists(name string) bool {
	return execCommand("podman", "container", "exists", name).Run() == nil
}

// printDryRun prints how a podman command would be invoked (without running it).
// Arguments with spaces or special characters are quoted for shell clarity.
func printDryRun(args []string) {
	quoted := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t\"'\\") {
			quoted[i] = fmt.Sprintf("%q", a)
		} else {
			quoted[i] = a
		}
	}
	fmt.Println("podman " + strings.Join(quoted, " "))
}

// connectContainer opens an interactive session in the running container via podman exec.
func connectContainer(name, command string, extra []string) error {
	args := append([]string{"exec", "-ti"}, extra...)
	args = append(args, name)
	args = append(args, strings.Fields(command)...)
	if err := runInteractive("podman", args...); err != nil {
		return fmt.Errorf("connect to container: %w", err)
	}
	return nil
}

// containerNameWithSuffix returns baseName with suffix appended if non-empty.
func containerNameWithSuffix(baseName, suffix string) string {
	if suffix == "" {
		return baseName
	}
	return baseName + suffix
}

// workspaceMountPath returns the container-side mount path for the current working directory.
func workspaceMountPath(cfg Config) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current working directory: %w", err)
	}
	dirName := filepath.Base(cwd)
	return fmt.Sprintf("/workspace/%s/%s", cfg.General.ID, dirName), nil
}

// containerArgs returns podman flags for container name, hostname, and basic settings.
// Security and capability args are stored in [create].arguments in silo.toml.
func containerArgs(cfg Config, containerNameSuffix ...string) []string {
	suffix := ""
	if len(containerNameSuffix) > 0 {
		suffix = containerNameSuffix[0]
	}
	containerName := containerNameWithSuffix(cfg.General.ContainerName, suffix)

	args := []string{"--name", containerName, "--hostname", containerName}
	args = append(args, "--user", cfg.General.User)

	return args
}

// buildContainerArgs returns podman container-specific arguments from cfg.
// Callers should prepend subcommands ("create", "run") as needed.
func buildContainerArgs(cfg Config) ([]string, error) {
	hostDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current working directory: %w", err)
	}

	var args []string

	args = append(args, containerArgs(cfg)...)

	// Workspace mount (host dir → container path)
	containerDir, err := workspaceMountPath(cfg)
	if err != nil {
		return nil, fmt.Errorf("get workspace mount path: %w", err)
	}
	args = append(args, "--volume", fmt.Sprintf("%s:%s:Z", hostDir, containerDir))
	args = append(args, "--workdir", containerDir)

	// Shared volume - mount each path as a subpath of the named volume
	for _, path := range cfg.SharedVolume.Paths {
		containerPath := resolveContainerPath(path, cfg.General.User)
		// Skip invalid/relative paths
		if containerPath == "" {
			continue
		}
		// subpath is the path within the volume (without leading /)
		subpath := strings.TrimPrefix(containerPath, "/")
		args = append(args, "--mount", fmt.Sprintf("type=volume,source=%s,target=%s,subpath=%s,Z", cfg.getSharedVolumeName(), containerPath, subpath))
	}

	return args, nil
}

// createContainer creates a new container. It does not start it.
// Extra args are forwarded to podman create.
func createContainer(cfg Config, extra []string) error {
	podmanArgs, err := buildContainerArgs(cfg)
	if err != nil {
		return fmt.Errorf("build container arguments: %w", err)
	}
	createArgs := append([]string{"create"}, podmanArgs...)
	createArgs = append(createArgs, extra...)
	createArgs = append(createArgs, cfg.General.ImageName)

	fmt.Printf("Creating %s...\n", cfg.General.ContainerName)
	if err := runVisible("podman", createArgs...); err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	return nil
}

// startContainer starts a stopped container.
func startContainer(name string) error {
	fmt.Printf("Starting %s...\n", name)
	if err := execCommand("podman", "start", name).Run(); err != nil {
		return fmt.Errorf("start container: %w", err)
	}
	return nil
}

// stopContainer stops a running container immediately.
func stopContainer(name string) error {
	fmt.Printf("Stopping %s...\n", name)
	if err := execCommand("podman", "stop", "-t", "0", name).Run(); err != nil {
		return fmt.Errorf("stop container: %w", err)
	}
	return nil
}

// --- ensure chain: ensureInit → ensureBuilt → ensureCreated → ensureStarted ---

// ensureInit initializes workspace config, workspace starter files, and
// user starter files. It delegates user-file creation to ensureUserFiles so
// `silo init` and `silo user init` share a single implementation.
// If podman is non-nil, .silo/home.nix will include module.podman.enable based on the value.
// If podman is nil, the podman setting seeded from silo.in.toml is preserved.
func ensureInit(podman *bool) (Config, bool, error) {
	cfg, firstRun, err := initWorkspaceConfig()
	if err != nil {
		return cfg, firstRun, fmt.Errorf("initialize workspace configuration: %w", err)
	}
	if err := ensureWorkspaceFiles(podman != nil && *podman); err != nil {
		return cfg, firstRun, fmt.Errorf("ensure workspace files: %w", err)
	}
	if err := ensureUserFiles(); err != nil {
		return cfg, firstRun, fmt.Errorf("ensure user files: %w", err)
	}
	if firstRun {
		if podman != nil {
			cfg.Features.Podman = *podman
		}
		var defaultArgs []string
		if cfg.Features.Podman {
			defaultArgs = []string{"--security-opt", "label=disable", "--device", "/dev/fuse"}
		} else {
			defaultArgs = []string{"--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges"}
		}
		cfg.Create.Arguments = append(cfg.Create.Arguments, defaultArgs...)
		if err := cfg.saveWorkspaceConfig(); err != nil {
			return cfg, firstRun, fmt.Errorf("save workspace config: %w", err)
		}
	}
	return cfg, firstRun, nil
}

// ensureBuilt ensures images exist, building them if needed.
func ensureBuilt() (Config, error) {
	cfg, _, err := ensureInit(nil)
	if err != nil {
		return cfg, fmt.Errorf("initialize workspace: %w", err)
	}
	if err := ensureImages(cfg); err != nil {
		return cfg, fmt.Errorf("ensure images: %w", err)
	}
	return cfg, nil
}

// ensureCreated ensures the container exists, creating it if needed.
func ensureCreated() (Config, error) {
	cfg, err := ensureBuilt()
	if err != nil {
		return cfg, fmt.Errorf("build images: %w", err)
	}
	if !containerExists(cfg.General.ContainerName) {
		if err := createContainer(cfg, cfg.Create.Arguments); err != nil {
			return cfg, fmt.Errorf("create container: %w", err)
		}
	}
	return cfg, nil
}

// ensureVolumeSetup ensures directories exist on the shared volume.
func ensureVolumeSetup(cfg Config) error {
	return volumeSetup(cfg)
}

// ensureStarted ensures the container is running, starting it if needed.
func ensureStarted() (Config, error) {
	cfg, err := ensureCreated()
	if err != nil {
		return cfg, fmt.Errorf("create container: %w", err)
	}
	if !containerRunning(cfg.General.ContainerName) {
		if err := ensureVolumeSetup(cfg); err != nil {
			return cfg, err
		}
		if err := startContainer(cfg.General.ContainerName); err != nil {
			return cfg, fmt.Errorf("start container: %w", err)
		}
	}
	return cfg, nil
}

// removeContainer forcibly removes the named container.
func removeContainer(name string) error {
	if err := runVisible("podman", "rm", "-f", name); err != nil {
		return fmt.Errorf("remove container: %w", err)
	}
	return nil
}

// removeImage removes the named image.
func removeImage(name string) error {
	if err := runVisible("podman", "rmi", name); err != nil {
		return fmt.Errorf("remove image: %w", err)
	}
	return nil
}

// runVisible runs a command with stdout and stderr connected to the terminal.
func runVisible(name string, args ...string) error {
	cmd := execCommand(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runInteractive runs a command with full stdio connected to the terminal.
func runInteractive(name string, args ...string) error {
	cmd := execCommand(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
