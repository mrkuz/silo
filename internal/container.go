package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var execCommand = exec.Command

const volumeMountPath = "/silo/shared"

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

// VolumeSetup creates directories on the silo-shared volume from the host side
// by running a temporary container with the user image, ensuring directories exist
// before they are mounted as subpath volumes.
func VolumeSetup(cfg Config) error {
	if !cfg.Features.SharedVolume || len(cfg.SharedVolume.Paths) == 0 {
		return nil
	}

	// Ensure the user image exists before using it for volume setup
	userImage := BaseImageName(cfg.General.User)
	if !ImageExists(userImage) {
		tc, err := NewTemplateContext(cfg)
		if err != nil {
			return fmt.Errorf("build template context: %w", err)
		}
		if err := EnsureUserImage(tc); err != nil {
			return fmt.Errorf("ensure user image: %w", err)
		}
	}

	var mkdirCmd strings.Builder
	for i, path := range cfg.SharedVolume.Paths {
		containerPath := ResolveContainerPath(path, cfg.General.User)
		if containerPath == "" {
			continue
		}
		if i > 0 {
			mkdirCmd.WriteString(" && ")
		}
		isDir := strings.HasSuffix(path, "/")
		volPath := volumeMountPath + containerPath
		if isDir {
			mkdirCmd.WriteString("mkdir -p " + volPath + " && chmod 755 " + volPath)
		} else {
			mkdirCmd.WriteString("mkdir -p $(dirname " + volPath + ") && touch " + volPath + " && chmod 644 " + volPath)
		}
	}
	cmd := execCommand("podman", "run", "--rm", "-v", cfg.GetSharedVolumeName()+":"+volumeMountPath+":Z", userImage, "sh", "-c", mkdirCmd.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("volume setup: %w", err)
	}
	return nil
}

// ContainerRunning checks if a container is currently running.
func ContainerRunning(name string) bool {
	out, err := execCommand("podman", "container", "inspect", "--format", "{{.State.Running}}", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// ContainerExists checks if a container exists in any state.
func ContainerExists(name string) bool {
	return execCommand("podman", "container", "exists", name).Run() == nil
}

// PrintDryRun prints how a podman command would be invoked (without running it).
// Arguments with spaces or special characters are quoted for shell clarity.
func PrintDryRun(args []string) {
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

// ConnectContainer opens an interactive session in the running container via podman exec.
func ConnectContainer(name, command string, extra []string) error {
	args := append([]string{"exec", "-ti"}, extra...)
	args = append(args, name)
	args = append(args, strings.Fields(command)...)
	if err := RunInteractive("podman", args...); err != nil {
		return fmt.Errorf("connect to container: %w", err)
	}
	return nil
}

// ContainerNameWithSuffix returns baseName with suffix appended if non-empty.
func ContainerNameWithSuffix(baseName, suffix string) string {
	if suffix == "" {
		return baseName
	}
	return baseName + suffix
}

// WorkspaceMountPath returns the container-side mount path for the current working directory.
func WorkspaceMountPath(cfg Config) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current working directory: %w", err)
	}
	dirName := filepath.Base(cwd)
	return fmt.Sprintf("/workspace/%s/%s", cfg.General.ID, dirName), nil
}

// ContainerArgs returns podman flags for container name, hostname, and basic settings.
// Security and capability args are stored in [create].arguments in silo.toml.
func ContainerArgs(cfg Config, containerNameSuffix ...string) []string {
	suffix := ""
	if len(containerNameSuffix) > 0 {
		suffix = containerNameSuffix[0]
	}
	containerName := ContainerNameWithSuffix(cfg.General.ContainerName, suffix)

	args := []string{"--name", containerName, "--hostname", containerName}
	args = append(args, "--user", cfg.General.User)

	return args
}

// BuildContainerArgs returns podman container-specific arguments from cfg.
// Callers should prepend subcommands ("create", "run") as needed.
func BuildContainerArgs(cfg Config) ([]string, error) {
	hostDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current working directory: %w", err)
	}

	var args []string

	args = append(args, ContainerArgs(cfg)...)

	// Workspace mount (host dir → container path)
	containerDir, err := WorkspaceMountPath(cfg)
	if err != nil {
		return nil, fmt.Errorf("get workspace mount path: %w", err)
	}
	args = append(args, "--volume", fmt.Sprintf("%s:%s:Z", hostDir, containerDir))
	args = append(args, "--workdir", containerDir)

	// Shared volume - mount each path as a subpath of the named volume
	for _, path := range cfg.SharedVolume.Paths {
		containerPath := ResolveContainerPath(path, cfg.General.User)
		// Skip invalid/relative paths
		if containerPath == "" {
			continue
		}
		// subpath is the path within the volume (without leading /)
		subpath := strings.TrimPrefix(containerPath, "/")
		args = append(args, "--mount", fmt.Sprintf("type=volume,source=%s,target=%s,subpath=%s,Z", cfg.GetSharedVolumeName(), containerPath, subpath))
	}

	return args, nil
}

// CreateContainer creates a new container. It does not start it.
// Extra args are forwarded to podman create.
func CreateContainer(cfg Config, extra []string) error {
	podmanArgs, err := BuildContainerArgs(cfg)
	if err != nil {
		return fmt.Errorf("build container arguments: %w", err)
	}
	createArgs := append([]string{"create"}, podmanArgs...)
	createArgs = append(createArgs, extra...)
	createArgs = append(createArgs, cfg.General.ImageName)

	fmt.Printf("Creating %s...\n", cfg.General.ContainerName)
	if err := RunVisible("podman", createArgs...); err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	return nil
}

// StartContainer starts a stopped container.
func StartContainer(name string) error {
	fmt.Printf("Starting %s...\n", name)
	if err := execCommand("podman", "start", name).Run(); err != nil {
		return fmt.Errorf("start container: %w", err)
	}
	return nil
}

// StopContainer stops a running container immediately.
func StopContainer(name string) error {
	fmt.Printf("Stopping %s...\n", name)
	if err := execCommand("podman", "stop", "-t", "0", name).Run(); err != nil {
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
	cmd := execCommand(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunInteractive runs a command with full stdio connected to the terminal.
func RunInteractive(name string, args ...string) error {
	cmd := execCommand(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PrintInitFileStatus prints the status of an init file.
func PrintInitFileStatus(path string) error {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("%s already exists\n", path)
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

// RemoveNamedContainer removes a named container, handling running state and force flag.
func RemoveNamedContainer(name string, force bool) error {
	if !ContainerExists(name) {
		PrintNotFound(name)
		return nil
	}
	if ContainerRunning(name) {
		if !force {
			return fmt.Errorf("%s is running", name)
		}
		if err := StopContainer(name); err != nil {
			return fmt.Errorf("stop container before removal: %w", err)
		}
	}
	fmt.Printf("Removing %s...\n", name)
	if err := RemoveContainer(name); err != nil {
		return fmt.Errorf("remove container: %w", err)
	}
	return nil
}
