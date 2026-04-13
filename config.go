package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/user"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const siloDir = ".silo"
const siloToml = ".silo/silo.toml"
const volumeMountPath = "/silo/shared"

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

// getSharedVolumeName returns the shared volume name, defaulting to "silo-shared".
func (c *Config) getSharedVolumeName() string {
	if c.SharedVolume.Name != "" {
		return c.SharedVolume.Name
	}
	return "silo-shared"
}

type ConnectConfig struct {
	Command string `toml:"command"`
}

type CreateConfig struct {
	Arguments []string `toml:"arguments"`
}

// defaultConfig returns a Config with a new random ID and current user.
func defaultConfig() (Config, error) {
	u, err := user.Current()
	if err != nil {
		return Config{}, fmt.Errorf("get current user: %w", err)
	}
	id := generateID()
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

// parseTOML decodes a TOML config file.
func parseTOML(path string) (Config, error) {
	var c Config
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	return c, nil
}

// requireWorkspaceConfig returns the workspace config or an error if .silo/silo.toml is missing.
func requireWorkspaceConfig() (Config, error) {
	if _, err := os.Stat(siloToml); os.IsNotExist(err) {
		return Config{}, fmt.Errorf("no .silo/silo.toml found — run 'silo init' to create it")
	}
	cfg, err := parseTOML(siloToml)
	if err != nil {
		return Config{}, fmt.Errorf("parse workspace configuration: %w", err)
	}
	return cfg, nil
}

// saveWorkspaceConfig persists the config to .silo/silo.toml.
func (c Config) saveWorkspaceConfig() error {
	if err := os.MkdirAll(siloDir, 0755); err != nil {
		return fmt.Errorf("create .silo directory: %w", err)
	}
	f, err := os.Create(siloToml)
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

// userConfigDir returns $XDG_CONFIG_HOME/silo (or ~/.config/silo by default).
func userConfigDir() (string, error) {
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
func ensureFile(path string, content []byte) error {
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

// ensureUserHomeNix creates $XDG_CONFIG_HOME/silo/home-user.nix if it does not exist.
func ensureUserHomeNix() error {
	dir, err := userConfigDir()
	if err != nil {
		return fmt.Errorf("create home-user.nix in config directory: %w", err)
	}
	return ensureFile(filepath.Join(dir, "home-user.nix"), []byte(homeUserNix))
}

// ensureWorkspaceHomeNix creates .silo/home.nix if it does not exist.
// If podman is true, module.podman.enable is set to true.
func ensureWorkspaceHomeNix(podman bool) error {
	content, err := renderWorkspaceHomeNix(podman)
	if err != nil {
		return fmt.Errorf("render workspace home.nix: %w", err)
	}
	return ensureFile(filepath.Join(siloDir, "home.nix"), []byte(content))
}

// ensureDevcontainerInJSON creates $XDG_CONFIG_HOME/silo/devcontainer.in.json if it does not exist.
func ensureDevcontainerInJSON() error {
	dir, err := userConfigDir()
	if err != nil {
		return fmt.Errorf("create devcontainer.in.json in config directory: %w", err)
	}
	return ensureFile(filepath.Join(dir, "devcontainer.in.json"), []byte("{}\n"))
}

// ensureSiloInTOML creates $XDG_CONFIG_HOME/silo/silo.in.toml if it does not exist.
func ensureSiloInTOML() error {
	dir, err := userConfigDir()
	if err != nil {
		return fmt.Errorf("create silo.in.toml in config directory: %w", err)
	}
	return ensureFile(filepath.Join(dir, "silo.in.toml"), []byte{})
}

// loadSiloInTOML parses $XDG_CONFIG_HOME/silo/silo.in.toml.
// The [general] section is not meaningful and is ignored.
// Returns an empty Config if the file does not exist.
func loadSiloInTOML() (Config, error) {
	dir, err := userConfigDir()
	if err != nil {
		return Config{}, fmt.Errorf("get config directory to load silo.in.toml: %w", err)
	}
	path := filepath.Join(dir, "silo.in.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Config{}, nil
	}
	return parseTOML(path)
}

// baseImageName returns the user image tag for the given user.
func baseImageName(user string) string {
	return "silo-" + user
}

// initWorkspaceConfig initializes workspace config from defaults or user settings.
// Returns (cfg, firstRun, error). On first run, cfg is built from defaults and silo.in.toml.
// On subsequent runs, cfg is loaded from silo.toml. Does NOT save — caller must save on first run.
func initWorkspaceConfig() (Config, bool, error) {
	var cfg Config
	if _, err := os.Stat(siloToml); os.IsNotExist(err) {
		// First run: seed from user config, fall back to built-in defaults.
		defaults, err := defaultConfig()
		if err != nil {
			return cfg, false, err
		}
		cfg, err = loadSiloInTOML()
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
		cfg, err = parseTOML(siloToml)
		if err != nil {
			return cfg, false, fmt.Errorf("load workspace silo.toml: %w", err)
		}
		return cfg, false, nil
	}
}

// ensureUserFiles silently creates user starter files if they do not exist.
// Shared by `silo init` (via ensureInit) and `silo user init` (via cmdUserInit,
// which wraps this with a warn-on-existing message).
func ensureUserFiles() error {
	if err := ensureUserHomeNix(); err != nil {
		return fmt.Errorf("ensure home-user.nix: %w", err)
	}
	if err := ensureDevcontainerInJSON(); err != nil {
		return fmt.Errorf("ensure devcontainer.in.json: %w", err)
	}
	if err := ensureSiloInTOML(); err != nil {
		return fmt.Errorf("ensure silo.in.toml: %w", err)
	}
	return nil
}

// ensureUserImage builds the shared user image if it does not exist.
func ensureUserImage(tc TemplateContext) error {
	userImage := tc.BaseImage
	if imageExists(userImage) {
		return nil
	}
	fmt.Printf("Building user image %s...\n", userImage)
	if err := buildUserImage(userImage, tc); err != nil {
		return fmt.Errorf("build user image: %w", err)
	}
	return nil
}

// ensureWorkspaceFiles silently creates workspace starter files if they do not exist.
func ensureWorkspaceFiles(podman bool) error {
	if err := ensureWorkspaceHomeNix(podman); err != nil {
		return fmt.Errorf("ensure workspace home.nix: %w", err)
	}
	return nil
}

// userStarterFile describes a single user-config starter file.
type userStarterFile struct {
	path    string
	content []byte
}

// userStarterFiles returns the list of user starter files that `silo user init` writes.
func userStarterFiles() ([]userStarterFile, error) {
	dir, err := userConfigDir()
	if err != nil {
		return nil, fmt.Errorf("get user config directory: %w", err)
	}
	return []userStarterFile{
		{filepath.Join(dir, "home-user.nix"), []byte(homeUserNix)},
		{filepath.Join(dir, "devcontainer.in.json"), []byte("{}\n")},
		{filepath.Join(dir, "silo.in.toml"), []byte{}},
	}, nil
}

// ensureImages builds the user and workspace images if they don't yet exist.
func ensureImages(cfg Config) error {
	tc, err := newTemplateContext(cfg)
	if err != nil {
		return fmt.Errorf("build template context: %w", err)
	}
	if err := ensureUserImage(tc); err != nil {
		return err
	}
	if imageExists(cfg.General.ImageName) {
		fmt.Printf("%s already exists\n", cfg.General.ImageName)
		return nil
	}
	fmt.Printf("Building workspace image %s...\n", cfg.General.ImageName)
	if err := buildWorkspaceImage(cfg.General.ImageName, tc); err != nil {
		return fmt.Errorf("build workspace image: %w", err)
	}
	return nil
}
