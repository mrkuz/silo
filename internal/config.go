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
	General      UserGeneralConfig  `toml:"general"`
	SharedVolume SharedVolumeConfig `toml:"shared_volume"`
	Podman       UserPodmanConfig   `toml:"podman"`
}

type UserGeneralConfig struct {
	User string `toml:"user"`
}

type UserPodmanConfig struct {
	CreateArgs []string `toml:"create_args"`
}

// WorkspaceConfig holds workspace-level configuration from .silo/silo.toml
type WorkspaceConfig struct {
	General      WorkspaceGeneralConfig `toml:"general"`
	Features     FeaturesConfig         `toml:"features"`
	SharedVolume SharedVolumeConfig     `toml:"shared_volume"`
	Podman       PodmanConfig           `toml:"podman"`
}

type WorkspaceGeneralConfig struct {
	ID string `toml:"id"`
}

// MergedConfig holds the combined configuration for runtime use.
// It merges workspace config with user config, with user values taking
// precedence where applicable.
type MergedConfig struct {
	User         string
	ID           string
	Features     FeaturesConfig
	SharedVolume SharedVolumeConfig
	Podman       PodmanConfig
}

type FeaturesConfig struct {
	Podman bool `toml:"podman"`
}

type SharedVolumeConfig struct {
	Paths []string `toml:"paths"`
}

type PodmanConfig struct {
	CreateArgs []string `toml:"create_args"`
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
		SharedVolume: SharedVolumeConfig{
			Paths: []string{},
		},
		Podman: PodmanConfig{CreateArgs: []string{}},
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

// RequireWorkspaceConfig returns the workspace config or an error if .silo/silo.toml is missing
// or if [general].id is empty.
func RequireWorkspaceConfig() (WorkspaceConfig, error) {
	if _, err := os.Stat(SiloToml()); os.IsNotExist(err) {
		return WorkspaceConfig{}, fmt.Errorf("no .silo/silo.toml found — run 'silo init' to create it")
	}
	var cfg WorkspaceConfig
	if err := ParseTOML(SiloToml(), &cfg); err != nil {
		return WorkspaceConfig{}, err
	}
	if cfg.General.ID == "" {
		return WorkspaceConfig{}, fmt.Errorf("[general].id is required in .silo/silo.toml")
	}
	return cfg, nil
}

// RequireMergedConfig returns the workspace config merged with user config.
// This is used by commands that need both workspace and user settings.
func RequireMergedConfig() (MergedConfig, error) {
	workspaceCfg, err := RequireWorkspaceConfig()
	if err != nil {
		return MergedConfig{}, err
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
	if c.SharedVolume.Paths == nil {
		c.SharedVolume.Paths = []string{}
	}
	if c.Podman.CreateArgs == nil {
		c.Podman.CreateArgs = []string{}
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

// EnsureUserHomeNix creates $XDG_CONFIG_HOME/silo/home.user.nix if it does not exist.
func EnsureUserHomeNix() error {
	dir, err := UserConfigDir()
	if err != nil {
		return fmt.Errorf("create home.user.nix in config directory: %w", err)
	}
	return EnsureFile(filepath.Join(dir, "home.user.nix"), []byte(HomeUserNix))
}

// EnsureWorkspaceHomeNix creates .silo/home.nix if it does not exist.
// If force is true, the file is always overwritten.
func EnsureWorkspaceHomeNix(podman bool, force bool) error {
	content, err := RenderWorkspaceHomeNix(podman)
	if err != nil {
		return fmt.Errorf("render workspace home.nix: %w", err)
	}
	path := filepath.Join(SiloDir(), "home.nix")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create directory for file: %w", err)
	}
	if force {
		return os.WriteFile(path, []byte(content), 0644)
	}
	return EnsureFile(path, []byte(content))
}

// EnsureDevcontainerInJSON creates $XDG_CONFIG_HOME/silo/devcontainer.user.json if it does not exist.
func EnsureDevcontainerInJSON() error {
	dir, err := UserConfigDir()
	if err != nil {
		return fmt.Errorf("create devcontainer.user.json in config directory: %w", err)
	}
	return EnsureFile(filepath.Join(dir, "devcontainer.user.json"), []byte(emptyJSON))
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
		return UserConfig{}, fmt.Errorf("silo.user.toml not found at %s", path)
	}
	var cfg UserConfig
	if err := ParseTOML(path, &cfg); err != nil {
		return UserConfig{}, err
	}
	if cfg.General.User == "" {
		return UserConfig{}, fmt.Errorf("[general].user is required in silo.user.toml")
	}
	return cfg, nil
}

// BaseImageName returns the user image tag for the given user.
func BaseImageName(user string) string {
	return "silo-" + user
}

// userStarterFile describes a single user-config starter file.
type UserStarterFile struct {
	Path    string
	Content []byte
}

// UserStarterFiles returns the list of user starter files that `silo user init` writes.
func UserStarterFiles() ([]UserStarterFile, error) {
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
	return []UserStarterFile{
		{filepath.Join(dir, "home.user.nix"), []byte(HomeUserNix)},
		{filepath.Join(dir, "devcontainer.user.json"), []byte(emptyJSON)},
		{filepath.Join(dir, "silo.user.toml"), []byte(userTomlContent)},
	}, nil
}

// MergeUserInto merges user config into workspace config.
// Returns a MergedConfig without modifying either source.
// Merge rules:
//   - features.podman:         workspace overrides user config
//   - podman.create_args:      user values prepended to workspace defaults
//   - shared_volume.paths:     user values prepended to workspace defaults
//   - id:                       workspace only
//   - user:                     user only
func MergeUserInto(workspace WorkspaceConfig, user UserConfig) MergedConfig {
	result := MergedConfig{
		User:         user.General.User,
		ID:           workspace.General.ID,
		Features:     workspace.Features,
		SharedVolume: workspace.SharedVolume,
		Podman:       workspace.Podman,
	}
	if len(user.Podman.CreateArgs) > 0 {
		result.Podman.CreateArgs = append(user.Podman.CreateArgs, workspace.Podman.CreateArgs...)
	}
	if len(user.SharedVolume.Paths) > 0 {
		result.SharedVolume.Paths = append(user.SharedVolume.Paths, workspace.SharedVolume.Paths...)
	}
	return result
}

// InitWorkspaceConfig initializes workspace config from defaults.
// Returns (cfg, firstRun, error). On first run, cfg is built from defaults.
// On subsequent runs, cfg is loaded from silo.toml. Does NOT save — caller must save on first run.
func InitWorkspaceConfig() (WorkspaceConfig, bool, error) {
	if _, err := os.Stat(SiloToml()); os.IsNotExist(err) {
		cfg, err := DefaultWorkspaceConfig()
		return cfg, true, err
	}
	var cfg WorkspaceConfig
	if err := ParseTOML(SiloToml(), &cfg); err != nil {
		return cfg, false, fmt.Errorf("load workspace silo.toml: %w", err)
	}
	if cfg.General.ID == "" {
		return cfg, false, fmt.Errorf("[general].id is required in .silo/silo.toml")
	}
	return cfg, false, nil
}

// EnsureUserFiles creates user starter files if they do not exist.
func EnsureUserFiles() error {
	files, err := UserStarterFiles()
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

// EnsureUserImage builds the shared user image if it does not exist.
// If force is true, the image is always rebuilt regardless of whether it exists.
func EnsureUserImage(tc TemplateContext, force bool) error {
	userImage := tc.BaseImage
	if !force && ImageExists(userImage) {
		return nil
	}
	fmt.Printf("Building user image %s...\n", userImage)
	if err := BuildUserImage(userImage, tc, force); err != nil {
		return fmt.Errorf("build user image: %w", err)
	}
	return nil
}

// EnsureWorkspaceFiles silently creates workspace starter files if they do not exist.
func EnsureWorkspaceFiles(podman bool) error {
	return EnsureWorkspaceHomeNix(podman, false)
}

// EnsureImages builds the user and workspace images if they don't yet exist.
// If force is true, workspace image is always rebuilt regardless of whether it exists.
func EnsureImages(cfg WorkspaceConfig, force bool) error {
	tc, err := NewTemplateContextFromWorkspace(cfg)
	if err != nil {
		return fmt.Errorf("build template context: %w", err)
	}
	if err := EnsureUserImage(tc, false); err != nil {
		return err
	}
	if !force && ImageExists(WorkspaceImageName(cfg.General.ID)) {
		return nil
	}
	fmt.Printf("Building workspace image %s...\n", WorkspaceImageName(cfg.General.ID))
	if err := BuildWorkspaceImage(WorkspaceImageName(cfg.General.ID), tc, force); err != nil {
		return fmt.Errorf("build workspace image: %w", err)
	}
	return nil
}

// DefaultCreateArgs returns the default container create arguments based on podman setting.
func DefaultCreateArgs(podman bool) []string {
	if podman {
		return []string{"--security-opt", "label=disable", "--device", "/dev/fuse"}
	}
	return []string{"--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges"}
}

// EnsureInit initializes workspace config, workspace starter files, and
// user starter files. It delegates user-file creation to EnsureUserFiles so
// `silo init` and `silo user init` share a single implementation.
// If podman is non-nil, .silo/home.nix will include silo.podman.enable based on the value.
func EnsureInit(podman *bool) (WorkspaceConfig, bool, error) {
	cfg, firstRun, err := InitWorkspaceConfig()
	if err != nil {
		return cfg, firstRun, fmt.Errorf("initialize workspace configuration: %w", err)
	}
	if err := EnsureWorkspaceFiles(podman != nil && *podman); err != nil {
		return cfg, firstRun, fmt.Errorf("ensure workspace files: %w", err)
	}
	if err := EnsureUserFiles(); err != nil {
		return cfg, firstRun, fmt.Errorf("ensure user files: %w", err)
	}
	if firstRun {
		if podman != nil {
			cfg.Features.Podman = *podman
		}
		cfg.Podman.CreateArgs = DefaultCreateArgs(cfg.Features.Podman)
		if err := cfg.SaveWorkspaceConfig(); err != nil {
			return cfg, firstRun, fmt.Errorf("save workspace config: %w", err)
		}
	}
	return cfg, firstRun, nil
}

// EnsureBuilt ensures images exist, building them if needed.
func EnsureBuilt() (WorkspaceConfig, error) {
	cfg, _, err := EnsureInit(nil)
	if err != nil {
		return cfg, fmt.Errorf("initialize workspace: %w", err)
	}
	if err := EnsureImages(cfg, false); err != nil {
		return cfg, fmt.Errorf("ensure images: %w", err)
	}
	return cfg, nil
}

// EnsureCreated ensures the container exists, creating it if needed.
func EnsureCreated() (MergedConfig, error) {
	if _, _, err := EnsureInit(nil); err != nil {
		return MergedConfig{}, fmt.Errorf("initialize workspace: %w", err)
	}
	mergedCfg, err := RequireMergedConfig()
	if err != nil {
		return MergedConfig{}, fmt.Errorf("load merged config: %w", err)
	}
	if ContainerExists(WorkspaceContainerName(mergedCfg.ID)) {
		return mergedCfg, nil
	}
	workspaceCfg, err := RequireWorkspaceConfig()
	if err != nil {
		return mergedCfg, fmt.Errorf("load workspace config: %w", err)
	}
	if err := EnsureImages(workspaceCfg, false); err != nil {
		return mergedCfg, fmt.Errorf("ensure images: %w", err)
	}
	if err := CreateContainer(mergedCfg, mergedCfg.Podman.CreateArgs); err != nil {
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
