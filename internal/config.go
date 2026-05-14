package internal

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/user"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// UserConfig holds user-level configuration from silo.user.toml
type UserConfig struct {
	General     UserGeneralConfig `toml:"general"`
	Persistence PersistenceConfig `toml:"persistence"`
	Podman      UserPodmanConfig  `toml:"podman"`
}

type UserGeneralConfig struct {
	User string `toml:"user"`
}

type UserPodmanConfig struct {
	CreateArgs []string      `toml:"create_args"`
	Network    NetworkConfig `toml:"network"`
}

// WorkspaceConfig holds workspace-level configuration from .silo/silo.toml
type WorkspaceConfig struct {
	General     WorkspaceGeneralConfig `toml:"general"`
	Features    FeaturesConfig         `toml:"features"`
	Persistence PersistenceConfig      `toml:"persistence"`
	Podman      PodmanConfig           `toml:"podman"`
	Network     NetworkConfig          `toml:"network"`
	Limits      LimitsConfig           `toml:"limits"`
}

type WorkspaceGeneralConfig struct {
	ID string `toml:"id"`
}

// MergedConfig holds the combined configuration for runtime use.
// It merges workspace config with user config, with user values taking
// precedence where applicable.
type MergedConfig struct {
	User        string
	ID          string
	Features    FeaturesConfig
	Persistence PersistenceConfig
	Podman      PodmanConfig
	Network     NetworkConfig
	Limits      LimitsConfig
}

type FeaturesConfig struct {
	Podman bool `toml:"podman"`
}

type PersistenceConfig struct {
	SharedPaths  []string `toml:"shared_paths"`
	PrivatePaths []string `toml:"private_paths"`
}

type PodmanConfig struct {
	CreateArgs []string `toml:"create_args"`
}

type NetworkConfig struct {
	Ports []string `toml:"ports"`
}

type LimitsConfig struct {
	CPUs      int `toml:"cpus"`
	Memory    int `toml:"memory"`
	Processes int `toml:"processes"`
}

// WorkspaceContainerName returns container name derived from id.
func WorkspaceContainerName(id string) string {
	return "silo-" + id
}

// WorkspaceImageName returns image name derived from id.
func WorkspaceImageName(id string) string {
	return "silo-" + id
}

const emptyJSON = "{}\n"

// SiloDir returns the workspace silo directory name.
func SiloDir() string {
	return ".silo"
}

// SiloToml returns the workspace config file path.
func SiloToml() string {
	return ".silo/silo.toml"
}

// defaultWorkspaceConfig returns a WorkspaceConfig with a new random ID.
func DefaultWorkspaceConfig() (WorkspaceConfig, error) {
	id := generatedIDFunc()
	return WorkspaceConfig{
		General: WorkspaceGeneralConfig{
			ID: id,
		},
		Features: FeaturesConfig{
			Podman: false,
		},
		Persistence: PersistenceConfig{
			SharedPaths:  []string{},
			PrivatePaths: []string{},
		},
		Podman:  PodmanConfig{CreateArgs: []string{}},
		Network: NetworkConfig{Ports: []string{}},
		Limits:  LimitsConfig{CPUs: 0, Memory: 0, Processes: 1024},
	}, nil
}

var generatedIDFunc = generateID

// generateID returns an 8-character random lowercase alphanumeric identifier.
func generateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[n.Int64()]
	}
	return string(b)
}

// ParseTOML decodes a TOML config file into the given struct.
// Strict mode rejects unknown keys that don't match any struct field.
func ParseTOML(path string, cfg interface{}) error {
	meta, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return fmt.Errorf("%s: unsupported keys: %v", filepath.Base(path), undecoded)
	}
	return nil
}

// WriteTOML encodes and writes cfg to path.
func WriteTOML(path string, cfg interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", filepath.Base(path), err)
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}
	return nil
}

// RequireMergedConfig returns the workspace config merged with user config.
// This is used by commands that need both workspace and user settings.
func RequireMergedConfig() (MergedConfig, error) {
	if _, err := os.Stat(SiloToml()); os.IsNotExist(err) {
		return MergedConfig{}, fmt.Errorf("'.silo/silo.toml' not found")
	}
	var workspaceCfg WorkspaceConfig
	if err := ParseTOML(SiloToml(), &workspaceCfg); err != nil {
		return MergedConfig{}, err
	}
	if workspaceCfg.General.ID == "" {
		return MergedConfig{}, fmt.Errorf("[general].id is required in '.silo/silo.toml'")
	}
	userCfg, err := LoadSiloUserTOML()
	if err != nil {
		return MergedConfig{}, fmt.Errorf("load user configuration: %w", err)
	}
	return MergeUserInto(workspaceCfg, userCfg), nil
}

// SaveWorkspaceConfig persists the config to .silo/silo.toml.
func (c WorkspaceConfig) SaveWorkspaceConfig() error {
	if err := os.MkdirAll(SiloDir(), 0755); err != nil {
		return fmt.Errorf("create .silo directory: %w", err)
	}
	f, err := os.Create(SiloToml())
	if err != nil {
		return fmt.Errorf("create .silo/silo.toml: %w", err)
	}
	defer f.Close()
	if c.Persistence.SharedPaths == nil {
		c.Persistence.SharedPaths = []string{}
	}
	if c.Persistence.PrivatePaths == nil {
		c.Persistence.PrivatePaths = []string{}
	}
	if c.Podman.CreateArgs == nil {
		c.Podman.CreateArgs = []string{}
	}
	if c.Network.Ports == nil {
		c.Network.Ports = []string{}
	}
	enc := toml.NewEncoder(f)
	enc.Indent = ""
	return enc.Encode(c)
}

// UserConfigDir returns $XDG_CONFIG_HOME/silo (or ~/.config/silo by default).
func UserConfigDir() (string, error) {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get user home directory: %w", err)
		}
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "silo"), nil
}

// ensureFile creates a file with content if it does not already exist.
func EnsureFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create directory for file: %w", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, content, 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}
	return nil
}

// EnsureDevcontainerUserJSON creates $XDG_CONFIG_HOME/silo/devcontainer.user.json if it does not exist.
func EnsureDevcontainerUserJSON() error {
	dir, err := UserConfigDir()
	if err != nil {
		return fmt.Errorf("create devcontainer.user.json in config directory: %w", err)
	}
	return EnsureFile(filepath.Join(dir, "devcontainer.user.json"), []byte(EmptyDevcontainerUserJSON))
}

// LoadSiloUserTOML parses $XDG_CONFIG_HOME/silo/silo.user.toml.
// Returns an error if [general].user is missing.
func LoadSiloUserTOML() (UserConfig, error) {
	dir, err := UserConfigDir()
	if err != nil {
		return UserConfig{}, fmt.Errorf("get config directory to load silo.user.toml: %w", err)
	}
	path := filepath.Join(dir, "silo.user.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return UserConfig{}, fmt.Errorf("'silo.user.toml' not found at '%s'", path)
	}
	var cfg UserConfig
	if err := ParseTOML(path, &cfg); err != nil {
		return UserConfig{}, err
	}
	if cfg.General.User == "" {
		return UserConfig{}, fmt.Errorf("[general].user is required in 'silo.user.toml'")
	}
	return cfg, nil
}

// UserFile describes a single user configuration file.
type UserFile struct {
	Path    string
	Content []byte
}

// UserFiles returns the list of user files that `silo user init` writes.
func UserFiles() ([]UserFile, error) {
	dir, err := UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("get user config directory: %w", err)
	}
	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}
	userTomlContent := fmt.Sprintf(`[general]
user = %q
`, u.Username)
	return []UserFile{
		{filepath.Join(dir, "home.user.nix"), []byte(HomeUserNix)},
		{filepath.Join(dir, "devcontainer.user.json"), []byte(EmptyDevcontainerUserJSON)},
		{filepath.Join(dir, "silo.user.toml"), []byte(userTomlContent)},
	}, nil
}

// MergeUserInto merges user config into workspace config.
// Returns a MergedConfig without modifying either source.
// Merge rules:
//   - features.podman:          workspace overrides user config
//   - podman.create_args:       user values prepended to workspace defaults
//   - persistence.shared_paths: user values prepended to workspace defaults
//   - network.ports:            user values prepended to workspace defaults
//   - id:                       workspace only
//   - user:                     user only
func MergeUserInto(workspace WorkspaceConfig, user UserConfig) MergedConfig {
	result := MergedConfig{
		User:        user.General.User,
		ID:          workspace.General.ID,
		Features:    workspace.Features,
		Persistence: workspace.Persistence,
		Podman:      workspace.Podman,
		Network:     workspace.Network,
		Limits:      workspace.Limits,
	}
	if len(user.Podman.CreateArgs) > 0 {
		result.Podman.CreateArgs = append(user.Podman.CreateArgs, workspace.Podman.CreateArgs...)
	}
	if len(user.Persistence.SharedPaths) > 0 {
		result.Persistence.SharedPaths = append(user.Persistence.SharedPaths, workspace.Persistence.SharedPaths...)
	}
	if len(user.Persistence.PrivatePaths) > 0 {
		result.Persistence.PrivatePaths = append(user.Persistence.PrivatePaths, workspace.Persistence.PrivatePaths...)
	}
	if len(user.Podman.Network.Ports) > 0 {
		result.Network.Ports = append(user.Podman.Network.Ports, workspace.Network.Ports...)
	}
	return result
}

// EnsureUserFiles creates user starter files if they do not exist.
func EnsureUserFiles() error {
	files, err := UserFiles()
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := os.MkdirAll(filepath.Dir(f.Path), 0755); err != nil {
			return fmt.Errorf("create directory for file: %w", err)
		}
		if err := EnsureFile(f.Path, f.Content); err != nil {
			return err
		}
	}
	return nil
}

// EnsureWorkspaceFiles silently creates workspace starter files if they do not exist.
func EnsureWorkspaceFiles(podman bool) error {
	content, err := RenderWorkspaceHomeNix(podman)
	if err != nil {
		return fmt.Errorf("render workspace home.nix: %w", err)
	}
	path := filepath.Join(SiloDir(), "home.nix")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create directory for file: %w", err)
	}
	return EnsureFile(path, []byte(content))
}

// EnsureInit initializes workspace config, workspace files, and
// user files. It delegates user-file creation to EnsureUserFiles so
// `silo init` and `silo user init` share a single implementation.
// If podman is non-nil, .silo/home.nix will include silo.podman.enable based on the value.
func EnsureInit(podman *bool) (WorkspaceConfig, error) {
	var cfg WorkspaceConfig
	var firstRun bool

	if _, err := os.Stat(SiloToml()); os.IsNotExist(err) {
		var err error
		cfg, err = DefaultWorkspaceConfig()
		if err != nil {
			return cfg, fmt.Errorf("initialize workspace configuration: %w", err)
		}
		firstRun = true
	} else {
		if err := ParseTOML(SiloToml(), &cfg); err != nil {
			return cfg, fmt.Errorf("load workspace silo.toml: %w", err)
		}
		if cfg.General.ID == "" {
			return cfg, fmt.Errorf("[general].id is required in '.silo/silo.toml'")
		}
		firstRun = false
	}

	if err := EnsureWorkspaceFiles(podman != nil && *podman); err != nil {
		return cfg, fmt.Errorf("ensure workspace files: %w", err)
	}
	if err := EnsureUserFiles(); err != nil {
		return cfg, fmt.Errorf("ensure user files: %w", err)
	}

	if firstRun {
		if podman != nil {
			cfg.Features.Podman = *podman
		}
		cfg.Podman.CreateArgs = DefaultCreateArgs(cfg.Features.Podman)
		if err := cfg.SaveWorkspaceConfig(); err != nil {
			return cfg, fmt.Errorf("save workspace config: %w", err)
		}
	}
	return cfg, nil
}

// EnsureBuild initializes the workspace and builds the image if needed.
func EnsureBuild() error {
	_, err := EnsureInit(nil)
	if err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}
	mergedCfg, err := RequireMergedConfig()
	if err != nil {
		return fmt.Errorf("load merged config: %w", err)
	}
	tc, err := NewTemplateContext(mergedCfg)
	if err != nil {
		return fmt.Errorf("build template context: %w", err)
	}
	imageName := WorkspaceImageName(mergedCfg.ID)
	if ImageExists(imageName) {
		return nil
	}
	if err := BuildImage(imageName, tc, false); err != nil {
		return fmt.Errorf("build image: %w", err)
	}
	return nil
}

// EnsureCreated ensures the container exists, creating it if needed.
func EnsureCreated() (MergedConfig, error) {
	if err := EnsureBuild(); err != nil {
		return MergedConfig{}, fmt.Errorf("build image: %w", err)
	}
	mergedCfg, err := RequireMergedConfig()
	if err != nil {
		return MergedConfig{}, fmt.Errorf("load merged config: %w", err)
	}
	if ContainerExists(WorkspaceContainerName(mergedCfg.ID)) {
		return mergedCfg, nil
	}
	if err := CreateContainer(mergedCfg); err != nil {
		return mergedCfg, fmt.Errorf("create container: %w", err)
	}
	return mergedCfg, nil
}

// EnsureStarted ensures the container is running, starting it if needed.
func EnsureStarted() (MergedConfig, error) {
	mergedCfg, err := EnsureCreated()
	if err != nil {
		return MergedConfig{}, fmt.Errorf("create container: %w", err)
	}
	if !ContainerRunning(WorkspaceContainerName(mergedCfg.ID)) {
		performed, err := VolumeSetup(mergedCfg)
		if err != nil {
			return MergedConfig{}, err
		}
		if performed {
			fmt.Println("Volume setup complete")
		}
		if err := StartContainer(WorkspaceContainerName(mergedCfg.ID)); err != nil {
			return MergedConfig{}, fmt.Errorf("start container: %w", err)
		}
	}
	return mergedCfg, nil
}
