package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var execCommand = exec.Command

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

// containerArgs returns podman flags for container name, hostname, and security options.
func containerArgs(cfg Config, containerNameSuffix ...string) []string {
	suffix := ""
	if len(containerNameSuffix) > 0 {
		suffix = containerNameSuffix[0]
	}
	containerName := containerNameWithSuffix(cfg.General.ContainerName, suffix)

	args := []string{"--name", containerName, "--hostname", containerName}
	if cfg.Features.Nested {
		return append(args, "--security-opt", "label=disable", "--device", "/dev/fuse")
	}
	return append(args, "--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges")
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
	args = append(args, "--user", cfg.General.User)

	// Workspace mount (host dir → container path)
	containerDir, err := workspaceMountPath(cfg)
	if err != nil {
		return nil, fmt.Errorf("get workspace mount path: %w", err)
	}
	args = append(args, "--volume", fmt.Sprintf("%s:%s:Z", hostDir, containerDir))
	args = append(args, "--workdir", containerDir)

	// Shared volume
	if cfg.Features.SharedVolume {
		args = append(args, "--volume", sharedVolumeName+":"+sharedVolumeMount+":Z")
	}

	return args, nil
}

// sharedPathEntry holds pre-computed source and destination paths for a shared volume symlink.
type sharedPathEntry struct {
	Src   string
	Dst   string
	IsDir bool
}

// buildSharedVolumeEntries converts path strings to entries for creating symlinks in the container.
// Paths ending with '/' are treated as directories; others as files.
// $HOME prefixes are expanded to ${HOME} for shell-time substitution.
func buildSharedVolumeEntries(paths []string) []sharedPathEntry {
	entries := make([]sharedPathEntry, 0, len(paths))
	for _, raw := range paths {
		isDir := len(raw) > 0 && raw[len(raw)-1] == '/'
		dst := strings.TrimRight(raw, "/")
		var src string
		if strings.HasPrefix(dst, "$HOME") {
			src = sharedVolumeMount + "${HOME}" + dst[len("$HOME"):]
		} else {
			src = sharedVolumeMount + dst
		}
		entries = append(entries, sharedPathEntry{Src: src, Dst: dst, IsDir: isDir})
	}
	return entries
}

// hasSharedPaths checks if shared volume is enabled with at least one path.
func hasSharedPaths(cfg Config) bool {
	return cfg.Features.SharedVolume && len(cfg.SharedVolume.Paths) > 0
}

const setupScriptPath = "/silo/setup.sh"

// setupContainer runs the setup script inside a running container.
// The script itself handles the setup-done marker.
func setupContainer(cfg Config) error {
	if !hasSharedPaths(cfg) {
		return nil
	}
	cmd := execCommand("podman", "exec", cfg.General.ContainerName, "bash", setupScriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("container setup: %w", err)
	}
	return nil
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

// --- ensure chain: ensureSetup → ensureStarted → ensureCreated → ensureBuilt → ensureInit ---

// ensureInit initializes workspace config, workspace starter files, and
// user starter files. It delegates user-file creation to ensureUserFiles so
// `silo init` and `silo user init` share a single implementation.
func ensureInit() (Config, bool, error) {
	cfg, firstRun, err := initWorkspaceConfig()
	if err != nil {
		return cfg, firstRun, fmt.Errorf("initialize workspace configuration: %w", err)
	}
	if err := ensureWorkspaceFiles(); err != nil {
		return cfg, firstRun, fmt.Errorf("ensure workspace files: %w", err)
	}
	if err := ensureUserFiles(); err != nil {
		return cfg, firstRun, fmt.Errorf("ensure user files: %w", err)
	}
	return cfg, firstRun, nil
}

// ensureBuilt ensures images exist, building them if needed.
func ensureBuilt() (Config, error) {
	cfg, _, err := ensureInit()
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
		if err := createContainer(cfg, cfg.Create.ExtraArgs); err != nil {
			return cfg, fmt.Errorf("create container: %w", err)
		}
	}
	return cfg, nil
}

// ensureStarted ensures the container is running, starting it if needed.
func ensureStarted() (Config, error) {
	cfg, err := ensureCreated()
	if err != nil {
		return cfg, fmt.Errorf("create container: %w", err)
	}
	if !containerRunning(cfg.General.ContainerName) {
		if err := startContainer(cfg.General.ContainerName); err != nil {
			return cfg, fmt.Errorf("start container: %w", err)
		}
	}
	return cfg, nil
}

// ensureSetup ensures the container is running and post-start setup has been applied.
func ensureSetup() (Config, error) {
	cfg, err := ensureStarted()
	if err != nil {
		return cfg, fmt.Errorf("start container: %w", err)
	}
	if err := setupContainer(cfg); err != nil {
		return cfg, fmt.Errorf("setup container: %w", err)
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
