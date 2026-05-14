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
	addPath := func(prefix string, path string) {
		containerPath := ResolveContainerPath(path, cfg.User)
		if containerPath == "" {
			return
		}
		if mkdirCmd.Len() > 0 {
			mkdirCmd.WriteString(" && ")
		}
		isDir := strings.HasSuffix(path, "/")
		volPath := prefix + containerPath
		if isDir {
			mkdirCmd.WriteString("mkdir -p " + volPath + " && chmod 755 " + volPath)
		} else {
			mkdirCmd.WriteString("mkdir -p $(dirname " + volPath + ") && touch " + volPath + " && chmod 644 " + volPath)
		}
	}

	for _, path := range cfg.Persistence.SharedPaths {
		addPath(volumeMountPath+"/shared", path)
	}
	for _, path := range cfg.Persistence.PrivatePaths {
		addPath(volumeMountPath+"/"+cfg.ID, path)
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

// DefaultCreateArgs returns podman security and capability arguments based on podman setting.
func DefaultCreateArgs(podman bool) []string {
	if podman {
		return []string{"--security-opt", "label=disable", "--device", "/dev/fuse"}
	}
	return []string{"--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges"}
}

// LimitsArgs returns podman resource limit arguments.
func LimitsArgs(limits LimitsConfig) []string {
	var args []string
	if limits.CPUs > 0 {
		args = append(args, fmt.Sprintf("--cpus=%d", limits.CPUs))
	} else {
		args = append(args, "--cpus=0")
	}
	if limits.Memory > 0 {
		args = append(args, fmt.Sprintf("--memory=%dm", limits.Memory))
	} else {
		args = append(args, "--memory=0")
	}
	if limits.Processes > 0 {
		args = append(args, fmt.Sprintf("--pids-limit=%d", limits.Processes))
	} else {
		args = append(args, "--pids-limit=-1")
	}
	return args
}

// ContainerArgs returns podman flags for container name, hostname, and basic settings.
// user is the username to set in the container; containerNameSuffix is an optional suffix for the container name.
func ContainerArgs(cfg WorkspaceConfig, containerNameSuffix string) []string {
	containerName := containerNameWithSuffix(WorkspaceContainerName(cfg.General.ID), containerNameSuffix)
	args := []string{"--name", containerName, "--hostname", containerName}
	return args
}

// CreateContainerArgs returns podman container-specific arguments from cfg.
func CreateContainerArgs(cfg MergedConfig) ([]string, error) {
	hostDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current working directory: %w", err)
	}

	args := append([]string{}, ContainerArgs(WorkspaceConfig{General: WorkspaceGeneralConfig{ID: cfg.ID}}, "")...)
	args = append(args, "--user", cfg.User)
	args = append(args, DefaultCreateArgs(cfg.Features.Podman)...)

	containerDir, err := WorkspaceMountPath(WorkspaceConfig{General: WorkspaceGeneralConfig{ID: cfg.ID}})
	if err != nil {
		return nil, fmt.Errorf("get workspace mount path: %w", err)
	}
	args = append(args, "--volume", fmt.Sprintf("%s:%s:z", hostDir, containerDir))
	args = append(args, "--workdir", containerDir)

	addVolume := func(subpath string, containerPath string) {
		if containerPath == "" {
			return
		}
		args = append(args, "--mount", fmt.Sprintf("type=volume,source=%s,target=%s,subpath=%s,z", "silo", containerPath, subpath))
	}

	for _, path := range cfg.Persistence.SharedPaths {
		containerPath := ResolveContainerPath(path, cfg.User)
		if cfg.User == "" {
			continue
		}
		addVolume("shared/"+strings.TrimPrefix(containerPath, "/"), containerPath)
	}
	for _, path := range cfg.Persistence.PrivatePaths {
		containerPath := ResolveContainerPath(path, cfg.User)
		if cfg.User == "" {
			continue
		}
		addVolume(cfg.ID+"/"+strings.TrimPrefix(containerPath, "/"), containerPath)
	}

	for _, port := range cfg.Network.Ports {
		args = append(args, "-p", port)
	}

	args = append(args, LimitsArgs(cfg.Limits)...)

	return args, nil
}

// CreateContainer creates a new container. It does not start it.
func CreateContainer(cfg MergedConfig) error {
	podmanArgs, err := CreateContainerArgs(cfg)
	if err != nil {
		return fmt.Errorf("create container arguments: %w", err)
	}
	createArgs := append([]string{"create"}, podmanArgs...)
	createArgs = append(createArgs, cfg.Podman.CreateArgs...)
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

// PrintRunningStatus prints the running status.
func PrintRunningStatus(isRunning bool) {
	if isRunning {
		fmt.Println("Running")
		return
	}
	fmt.Println("Stopped")
}
