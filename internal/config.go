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

// Config holds all persisted silo workspace configuration.
type Config struct {
	General      GeneralConfig      `toml:"general"`
	Features     FeaturesConfig     `toml:"features"`
	SharedVolume SharedVolumeConfig `toml:"shared_volume"`
	Connect      ConnectConfig      `toml:"connect"`
	Create       CreateConfig       `toml:"create"`
}

type GeneralConfig struct {
	ID            string `toml:"id"`
	User          string `toml:"user"`
	ContainerName string `toml:"container_name"`
	ImageName     string `toml:"image_name"`
}

type FeaturesConfig struct {
	SharedVolume bool `toml:"shared_volume"`
	Podman       bool `toml:"podman"`
}

type SharedVolumeConfig struct {
	Name  string   `toml:"name"`
	Paths []string `toml:"paths"`
}

type ConnectConfig struct {
	Command string `toml:"command"`
}

type CreateConfig struct {
	Arguments []string `toml:"arguments"`
}

const emptyJSON = "{}\n"

// GetSharedVolumeName returns the shared volume name, defaulting to "silo-shared".
func (c *Config) GetSharedVolumeName() string {
	if c.SharedVolume.Name != "" {
		return c.SharedVolume.Name
	}
	return "silo-shared"
}

// SiloDir returns the workspace silo directory name.
func SiloDir() string {
	return ".silo"
}

// SiloToml returns the workspace config file path.
func SiloToml() string {
	return ".silo/silo.toml"
}

// defaultConfig returns a Config with a new random ID and current user.
func DefaultConfig() (Config, error) {
	u, err := user.Current()
	if err != nil {
		return Config{}, fmt.Errorf("get current user: %w", err)
	}
	id := generatedIDFunc()
	return Config{
		General: GeneralConfig{
			ID:            id,
			User:          u.Username,
			ContainerName: "silo-" + id,
			ImageName:     "silo-" + id,
		},
		Features: FeaturesConfig{
			SharedVolume: false,
			Podman:       false,
		},
		SharedVolume: SharedVolumeConfig{
			Name:  "silo-shared",
			Paths: []string{},
		},
		Connect: ConnectConfig{
			Command: "/bin/sh",
		},
		Create: CreateConfig{
			Arguments: []string{},
		},
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

// ParseTOML decodes a TOML config file.
func ParseTOML(path string) (Config, error) {
	var c Config
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	return c, nil
}

// WriteTOML encodes and writes cfg to path.
func WriteTOML(path string, cfg Config) error {
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

// RequireWorkspaceConfig returns the workspace config or an error if .silo/silo.toml is missing.
func RequireWorkspaceConfig() (Config, error) {
	if _, err := os.Stat(SiloToml()); os.IsNotExist(err) {
		return Config{}, fmt.Errorf("no .silo/silo.toml found — run 'silo init' to create it")
	}
	cfg, err := ParseTOML(SiloToml())
	if err != nil {
		return Config{}, fmt.Errorf("parse workspace configuration: %w", err)
	}
	return cfg, nil
}

// SaveWorkspaceConfig persists the config to .silo/silo.toml.
func (c Config) SaveWorkspaceConfig() error {
	if err := os.MkdirAll(SiloDir(), 0755); err != nil {
		return fmt.Errorf("create .silo directory: %w", err)
	}
	f, err := os.Create(SiloToml())
	if err != nil {
		return fmt.Errorf("create .silo/silo.toml: %w", err)
	}
	defer f.Close()
	if c.SharedVolume.Name == "" {
		c.SharedVolume.Name = "silo-shared"
	}
	if c.SharedVolume.Paths == nil {
		c.SharedVolume.Paths = []string{}
	}
	if c.Create.Arguments == nil {
		c.Create.Arguments = []string{}
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

// EnsureUserHomeNix creates $XDG_CONFIG_HOME/silo/home-user.nix if it does not exist.
func EnsureUserHomeNix() error {
	dir, err := UserConfigDir()
	if err != nil {
		return fmt.Errorf("create home-user.nix in config directory: %w", err)
	}
	return EnsureFile(filepath.Join(dir, "home-user.nix"), []byte(HomeUserNix))
}

// EnsureWorkspaceHomeNix creates .silo/home.nix if it does not exist.
// If podman is true, module.podman.enable is set to true.
func EnsureWorkspaceHomeNix(podman bool) error {
	content, err := RenderWorkspaceHomeNix(podman)
	if err != nil {
		return fmt.Errorf("render workspace home.nix: %w", err)
	}
	return EnsureFile(filepath.Join(SiloDir(), "home.nix"), []byte(content))
}

// EnsureDevcontainerInJSON creates $XDG_CONFIG_HOME/silo/devcontainer.in.json if it does not exist.
func EnsureDevcontainerInJSON() error {
	dir, err := UserConfigDir()
	if err != nil {
		return fmt.Errorf("create devcontainer.in.json in config directory: %w", err)
	}
	return EnsureFile(filepath.Join(dir, "devcontainer.in.json"), []byte(emptyJSON))
}

// EnsureSiloInTOML creates $XDG_CONFIG_HOME/silo/silo.in.toml if it does not exist.
func EnsureSiloInTOML() error {
	dir, err := UserConfigDir()
	if err != nil {
		return fmt.Errorf("create silo.in.toml in config directory: %w", err)
	}
	return EnsureFile(filepath.Join(dir, "silo.in.toml"), []byte{})
}

// LoadSiloInTOML parses $XDG_CONFIG_HOME/silo/silo.in.toml.
// The [general] section is not meaningful and is ignored.
// Returns an empty Config if the file does not exist.
func LoadSiloInTOML() (Config, error) {
	dir, err := UserConfigDir()
	if err != nil {
		return Config{}, fmt.Errorf("get config directory to load silo.in.toml: %w", err)
	}
	path := filepath.Join(dir, "silo.in.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Config{}, nil
	}
	return ParseTOML(path)
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
	return []UserStarterFile{
		{filepath.Join(dir, "home-user.nix"), []byte(HomeUserNix)},
		{filepath.Join(dir, "devcontainer.in.json"), []byte(emptyJSON)},
		{filepath.Join(dir, "silo.in.toml"), []byte{}},
	}, nil
}

// InitWorkspaceConfig initializes workspace config from defaults or user settings.
// Returns (cfg, firstRun, error). On first run, cfg is built from defaults and silo.in.toml.
// On subsequent runs, cfg is loaded from silo.toml. Does NOT save — caller must save on first run.
func InitWorkspaceConfig() (Config, bool, error) {
	var cfg Config
	if _, err := os.Stat(SiloToml()); os.IsNotExist(err) {
		// First run: seed from user config, fall back to built-in defaults.
		defaults, err := DefaultConfig()
		if err != nil {
			return cfg, false, err
		}
		cfg, err = LoadSiloInTOML()
		if err != nil {
			return cfg, false, fmt.Errorf("load user silo.in.toml: %w", err)
		}
		cfg.General = defaults.General
		if cfg.Connect.Command == "" {
			cfg.Connect.Command = defaults.Connect.Command
		}
		if cfg.Features == (FeaturesConfig{}) {
			cfg.Features = defaults.Features
		}
		return cfg, true, nil
	} else {
		// Subsequent runs: use workspace config as-is.
		var err error
		cfg, err = ParseTOML(SiloToml())
		if err != nil {
			return cfg, false, fmt.Errorf("load workspace silo.toml: %w", err)
		}
		return cfg, false, nil
	}
}

// EnsureUserFiles silently creates user starter files if they do not exist.
func EnsureUserFiles() error {
	if err := EnsureUserHomeNix(); err != nil {
		return fmt.Errorf("ensure home-user.nix: %w", err)
	}
	if err := EnsureDevcontainerInJSON(); err != nil {
		return fmt.Errorf("ensure devcontainer.in.json: %w", err)
	}
	if err := EnsureSiloInTOML(); err != nil {
		return fmt.Errorf("ensure silo.in.toml: %w", err)
	}
	return nil
}

// EnsureUserImage builds the shared user image if it does not exist.
func EnsureUserImage(tc TemplateContext) error {
	userImage := tc.BaseImage
	if ImageExists(userImage) {
		return nil
	}
	fmt.Printf("Building user image %s...\n", userImage)
	if err := BuildUserImage(userImage, tc); err != nil {
		return fmt.Errorf("build user image: %w", err)
	}
	return nil
}

// EnsureWorkspaceFiles silently creates workspace starter files if they do not exist.
func EnsureWorkspaceFiles(podman bool) error {
	if err := EnsureWorkspaceHomeNix(podman); err != nil {
		return fmt.Errorf("ensure workspace home.nix: %w", err)
	}
	return nil
}

// EnsureImages builds the user and workspace images if they don't yet exist.
func EnsureImages(cfg Config) error {
	tc, err := NewTemplateContext(cfg)
	if err != nil {
		return fmt.Errorf("build template context: %w", err)
	}
	if err := EnsureUserImage(tc); err != nil {
		return err
	}
	if ImageExists(cfg.General.ImageName) {
		fmt.Printf("%s already exists\n", cfg.General.ImageName)
		return nil
	}
	fmt.Printf("Building workspace image %s...\n", cfg.General.ImageName)
	if err := BuildWorkspaceImage(cfg.General.ImageName, tc); err != nil {
		return fmt.Errorf("build workspace image: %w", err)
	}
	return nil
}

// EnsureInit initializes workspace config, workspace starter files, and
// user starter files. It delegates user-file creation to EnsureUserFiles so
// `silo init` and `silo user init` share a single implementation.
// If podman is non-nil, .silo/home.nix will include module.podman.enable based on the value.
// If podman is nil, the podman setting seeded from silo.in.toml is preserved.
func EnsureInit(podman *bool) (Config, bool, error) {
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
		var defaultArgs []string
		if cfg.Features.Podman {
			defaultArgs = []string{"--security-opt", "label=disable", "--device", "/dev/fuse"}
		} else {
			defaultArgs = []string{"--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges"}
		}
		cfg.Create.Arguments = append(cfg.Create.Arguments, defaultArgs...)
		if err := cfg.SaveWorkspaceConfig(); err != nil {
			return cfg, firstRun, fmt.Errorf("save workspace config: %w", err)
		}
	}
	return cfg, firstRun, nil
}

// EnsureBuilt ensures images exist, building them if needed.
func EnsureBuilt() (Config, error) {
	cfg, _, err := EnsureInit(nil)
	if err != nil {
		return cfg, fmt.Errorf("initialize workspace: %w", err)
	}
	if err := EnsureImages(cfg); err != nil {
		return cfg, fmt.Errorf("ensure images: %w", err)
	}
	return cfg, nil
}

// EnsureCreated ensures the container exists, creating it if needed.
func EnsureCreated() (Config, error) {
	cfg, err := EnsureBuilt()
	if err != nil {
		return cfg, fmt.Errorf("build images: %w", err)
	}
	if !ContainerExists(cfg.General.ContainerName) {
		if err := CreateContainer(cfg, cfg.Create.Arguments); err != nil {
			return cfg, fmt.Errorf("create container: %w", err)
		}
	}
	return cfg, nil
}

// EnsureStarted ensures the container is running, starting it if needed.
func EnsureStarted() (Config, error) {
	cfg, err := EnsureCreated()
	if err != nil {
		return cfg, fmt.Errorf("create container: %w", err)
	}
	if !ContainerRunning(cfg.General.ContainerName) {
		if err := VolumeSetup(cfg); err != nil {
			return cfg, err
		}
		if err := StartContainer(cfg.General.ContainerName); err != nil {
			return cfg, fmt.Errorf("start container: %w", err)
		}
	}
	return cfg, nil
}
