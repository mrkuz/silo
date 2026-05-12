package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const volumeMountPath = "/silo/persistence"

// ResolveContainerPath converts a shared volume path to its container mount target.
// - Absolute paths (e.g., "/etc/shared/") are used directly.
// - $HOME prefix (e.g., "$HOME/.cache/uv/") expands to "/home/<user>/.cache/uv".
// - Relative paths are not supported - a warning is printed and the path is skipped.
func ResolveContainerPath(path string, user string) string {
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

// VolumeSetup creates directories on the silo volume from the host side
// by running a temporary container with the workspace image, ensuring directories exist
// before they are mounted as subpath volumes. Returns true if directories were created.
func VolumeSetup(cfg MergedConfig) (bool, error) {
	if len(cfg.Persistence.SharedPaths) == 0 && len(cfg.Persistence.PrivatePaths) == 0 {
		return false, nil
	}

	workspaceImage := WorkspaceImageName(cfg.ID)
	if !ImageExists(workspaceImage) {
		return false, fmt.Errorf("workspace image %s not found", workspaceImage)
	}

	var mkdirCmd strings.Builder
	for i, path := range cfg.Persistence.SharedPaths {
		containerPath := ResolveContainerPath(path, cfg.User)
		if containerPath == "" {
			continue
		}
		if i > 0 {
			mkdirCmd.WriteString(" && ")
		}
		isDir := strings.HasSuffix(path, "/")
		volPath := volumeMountPath + "/shared" + containerPath
		if isDir {
			mkdirCmd.WriteString("mkdir -p " + volPath + " && chmod 755 " + volPath)
		} else {
			mkdirCmd.WriteString("mkdir -p $(dirname " + volPath + ") && touch " + volPath + " && chmod 644 " + volPath)
		}
	}

	for i, path := range cfg.Persistence.PrivatePaths {
		containerPath := ResolveContainerPath(path, cfg.User)
		if containerPath == "" {
			continue
		}
		if len(cfg.Persistence.SharedPaths) > 0 || i > 0 {
			mkdirCmd.WriteString(" && ")
		}
		isDir := strings.HasSuffix(path, "/")
		volPath := volumeMountPath + "/" + cfg.ID + containerPath
		if isDir {
			mkdirCmd.WriteString("mkdir -p " + volPath + " && chmod 755 " + volPath)
		} else {
			mkdirCmd.WriteString("mkdir -p $(dirname " + volPath + ") && touch " + volPath + " && chmod 644 " + volPath)
		}
	}
	cmd := ExecCommand("podman", "run", "--rm", "-v", "silo:"+volumeMountPath+":z", workspaceImage, "sh", "-c", mkdirCmd.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("volume setup: %w", err)
	}
	return true, nil
}

// ContainerRunning checks if a container is currently running.
func ContainerRunning(name string) bool {
	out, err := ExecCommand("podman", "container", "inspect", "--format", "{{.State.Running}}", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// ContainerExists checks if a container exists in any state.
func ContainerExists(name string) bool {
	return ExecCommand("podman", "container", "exists", name).Run() == nil
}

// ConnectContainer opens an interactive session in the running container via podman exec.
func ConnectContainer(name string) error {
	args := append([]string{"exec", "-ti"}, name)
	args = append(args, "sh", "-c", "$HOME/.nix-profile/bin/default-shell")
	if err := RunInteractive("podman", args...); err != nil {
		return fmt.Errorf("connect to container: %w", err)
	}
	return nil
}

// WorkspaceMountPath returns the container-side mount path for the current working directory.
func WorkspaceMountPath(cfg WorkspaceConfig) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current working directory: %w", err)
	}
	dirName := filepath.Base(cwd)
	return fmt.Sprintf("/workspace/%s/%s", cfg.General.ID, dirName), nil
}

// containerNameWithSuffix returns baseName with suffix appended if non-empty.
func containerNameWithSuffix(baseName, suffix string) string {
	if suffix == "" {
		return baseName
	}
	return baseName + suffix
}

// ContainerArgs returns podman flags for container name, hostname, and basic settings.
// Security and capability args are stored in [podman].create_args in silo.toml.
// user is the username to set in the container; containerNameSuffix is an optional suffix for the container name.
func ContainerArgs(cfg WorkspaceConfig, user string, containerNameSuffix ...string) []string {
	suffix := ""
	if len(containerNameSuffix) > 0 {
		suffix = containerNameSuffix[0]
	}
	containerName := containerNameWithSuffix(WorkspaceContainerName(cfg.General.ID), suffix)

	args := []string{"--name", containerName, "--hostname", containerName}
	if user != "" {
		args = append(args, "--user", user)
	}

	return args
}

// BuildContainerArgs returns podman container-specific arguments from cfg.
// Callers should prepend subcommands ("create", "run") as needed.
// user is the username for resolving container paths; if empty, shared volume paths are skipped.
func BuildContainerArgs(cfg WorkspaceConfig, user string) ([]string, error) {
	hostDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current working directory: %w", err)
	}

	var args []string

	args = append(args, ContainerArgs(cfg, user)...)

	// Workspace mount (host dir → container path)
	containerDir, err := WorkspaceMountPath(cfg)
	if err != nil {
		return nil, fmt.Errorf("get workspace mount path: %w", err)
	}
	args = append(args, "--volume", fmt.Sprintf("%s:%s:z", hostDir, containerDir))
	args = append(args, "--workdir", containerDir)

	// Shared volume - mount each path as a subpath of the named volume
	// user may be empty during container create; VolumeSetup will handle path creation
	for _, path := range cfg.Persistence.SharedPaths {
		if user == "" {
			continue
		}
		containerPath := ResolveContainerPath(path, user)
		// Skip invalid/relative paths
		if containerPath == "" {
			continue
		}
		// subpath is the path within the volume (without leading /)
		// paths now reside under /silo/persistence/shared
		subpath := "shared/" + strings.TrimPrefix(containerPath, "/")
		args = append(args, "--mount", fmt.Sprintf("type=volume,source=%s,target=%s,subpath=%s,z", "silo", containerPath, subpath))
	}

	// Private volume - mount each path as a subpath of the named volume
	for _, path := range cfg.Persistence.PrivatePaths {
		if user == "" {
			continue
		}
		containerPath := ResolveContainerPath(path, user)
		if containerPath == "" {
			continue
		}
		subpath := cfg.General.ID + "/" + strings.TrimPrefix(containerPath, "/")
		args = append(args, "--mount", fmt.Sprintf("type=volume,source=%s,target=%s,subpath=%s,z", "silo", containerPath, subpath))
	}

	for _, port := range cfg.Network.Ports {
		args = append(args, "-p", port)
	}

	return args, nil
}

// CreateContainer creates a new container. It does not start it.
// Extra args are forwarded to podman create.
func CreateContainer(cfg MergedConfig, extra []string) error {
	podmanArgs, err := BuildContainerArgs(WorkspaceConfig{
		General:     WorkspaceGeneralConfig{ID: cfg.ID},
		Features:    cfg.Features,
		Persistence: cfg.Persistence,
		Podman:      cfg.Podman,
		Network:     cfg.Network,
	}, cfg.User)
	if err != nil {
		return fmt.Errorf("build container arguments: %w", err)
	}
	createArgs := append([]string{"create"}, podmanArgs...)
	createArgs = append(createArgs, extra...)
	createArgs = append(createArgs, WorkspaceImageName(cfg.ID))

	fmt.Printf("Creating %s...\n", WorkspaceContainerName(cfg.ID))
	if err := RunVisible("podman", createArgs...); err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	return nil
}

// StartContainer starts a stopped container.
func StartContainer(name string) error {
	fmt.Printf("Starting %s...\n", name)
	if err := ExecCommand("podman", "start", name).Run(); err != nil {
		return fmt.Errorf("start container: %w", err)
	}
	return nil
}

// StopContainer stops a running container immediately.
func StopContainer(name string) error {
	fmt.Printf("Stopping %s...\n", name)
	if err := ExecCommand("podman", "stop", "-t", "0", name).Run(); err != nil {
		return fmt.Errorf("stop container: %w", err)
	}
	return nil
}

// RemoveContainer forcibly removes the named container.
func RemoveContainer(name string) error {
	if err := RunVisible("podman", "rm", "-f", name); err != nil {
		return fmt.Errorf("remove container: %w", err)
	}
	return nil
}

// RemoveImage removes the named image.
func RemoveImage(name string) error {
	if err := RunVisible("podman", "rmi", name); err != nil {
		return fmt.Errorf("remove image: %w", err)
	}
	return nil
}

// RunVisible runs a command with stdout and stderr connected to the terminal.
func RunVisible(name string, args ...string) error {
	cmd := ExecCommand(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunInteractive runs a command with full stdio connected to the terminal.
func RunInteractive(name string, args ...string) error {
	cmd := ExecCommand(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PrintInitFileStatus prints the status of an init file.
func PrintInitFileStatus(path string) error {
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

// PrintRunningStatus prints the running status.
func PrintRunningStatus(isRunning bool) {
	if isRunning {
		fmt.Println("Running")
		return
	}
	fmt.Println("Stopped")
}

// PrintNotFound prints a not found message.
func PrintNotFound(name string) {
	fmt.Printf("%s not found\n", name)
}
