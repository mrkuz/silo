package main

import (
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
// Arguments containing spaces are quoted for clarity.
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

// connectContainer opens an interactive session inside the running container via podman exec.
// Extra args are inserted before the container name as podman exec flags.
func connectContainer(name, command string, extra []string) error {
	args := append([]string{"exec", "-ti"}, extra...)
	args = append(args, name)
	args = append(args, strings.Fields(command)...)
	return runInteractive("podman", args...)
}

// containerNameWithSuffix appends at most one optional suffix to baseName.
func containerNameWithSuffix(baseName string, suffix ...string) string {
	if len(suffix) > 0 {
		return baseName + suffix[0]
	}
	return baseName
}

// containerArgs returns common podman container flags for name/hostname and security options.
// containerNameSuffix is appended to cfg.General.ContainerName; default is "".
func containerArgs(cfg Config, containerNameSuffix ...string) []string {
	containerName := containerNameWithSuffix(cfg.General.ContainerName, containerNameSuffix...)

	args := []string{"--name", containerName, "--hostname", containerName}
	if cfg.Features.Nested {
		return append(args, "--security-opt", "label=disable", "--device", "/dev/fuse")
	}
	return append(args, "--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges")
}

// buildContainerArgs constructs the podman container-specific arguments from cfg,
// without any subcommand prefix. Callers prepend "run", "-d" or "create" as needed.
func buildContainerArgs(cfg Config) ([]string, error) {
	hostDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	var args []string

	args = append(args, containerArgs(cfg)...)
	args = append(args, "--user", cfg.General.User)

	// Workspace mount (host dir → container path)
	if cfg.Features.Workspace {
		dirName := filepath.Base(hostDir)
		containerDir := fmt.Sprintf("/workspace/%s/%s", cfg.General.ID, dirName)
		args = append(args, "--volume", fmt.Sprintf("%s:%s:Z", hostDir, containerDir))
		args = append(args, "--workdir", containerDir)
	}

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

// buildSharedVolumeEntries converts raw path strings into pre-computed entries.
// Paths ending in '/' are directories; others are files.
// $HOME prefixes are expanded to ${HOME} for shell runtime expansion.
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

// hasSharedPaths reports whether the config enables the shared volume with at least one path.
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
		return err
	}
	createArgs := append([]string{"create"}, podmanArgs...)
	createArgs = append(createArgs, extra...)
	createArgs = append(createArgs, cfg.General.ImageName)

	fmt.Printf("Creating container %s...\n", cfg.General.ContainerName)
	if err := runVisible("podman", createArgs...); err != nil {
		return err
	}
	return nil
}

// startContainer starts a stopped container.
func startContainer(name string) error {
	fmt.Printf("Starting %s...\n", name)
	return execCommand("podman", "start", name).Run()
}

// stopContainer stops a running container immediately.
func stopContainer(name string) error {
	fmt.Printf("Stopping %s...\n", name)
	return execCommand("podman", "stop", "-t", "0", name).Run()
}

// --- ensure chain: ensureSetup → ensureStarted → ensureCreated → ensureBuilt → ensureInit ---

// ensureInit initializes workspace config and scaffold files.
func ensureInit() (Config, error) {
	cfg, err := initWorkspaceConfig()
	if err != nil {
		return cfg, err
	}
	return cfg, ensureScaffoldFiles()
}

// ensureBuilt ensures images exist, building them if needed.
func ensureBuilt() (Config, error) {
	cfg, err := ensureInit()
	if err != nil {
		return cfg, err
	}
	return cfg, ensureImages(cfg)
}

// ensureCreated ensures the container exists, creating it if needed.
func ensureCreated() (Config, error) {
	cfg, err := ensureBuilt()
	if err != nil {
		return cfg, err
	}
	if !containerExists(cfg.General.ContainerName) {
		if err := createContainer(cfg, cfg.Create.ExtraArgs); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

// ensureStarted ensures the container is running, starting it if needed.
func ensureStarted() (Config, error) {
	cfg, err := ensureCreated()
	if err != nil {
		return cfg, err
	}
	if !containerRunning(cfg.General.ContainerName) {
		if err := startContainer(cfg.General.ContainerName); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

// ensureSetup ensures the container is running and post-start setup has been applied.
func ensureSetup() (Config, error) {
	cfg, err := ensureStarted()
	if err != nil {
		return cfg, err
	}
	return cfg, setupContainer(cfg)
}

// removeContainer forcibly removes the named container.
func removeContainer(name string) error {
	return runVisible("podman", "rm", "-f", name)
}

// removeImage removes the named image.
func removeImage(name string) error {
	return runVisible("podman", "rmi", name)
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
