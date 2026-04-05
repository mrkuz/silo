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
const sharedVolumeName = "silo-shared"
const sharedVolumeMount = "/silo/shared"

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
	Workspace    bool `toml:"workspace"`
	SharedVolume bool `toml:"shared_volume"`
	Nested       bool `toml:"nested"`
}

type SharedVolumeConfig struct {
	Paths []string `toml:"paths"`
}

type ConnectConfig struct {
	Command string `toml:"command"`
}

type CreateConfig struct {
	ExtraArgs []string `toml:"extra_args"`
}

// defaultConfig returns a fresh Config with a new random ID and current user.
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
			Workspace:    true,
			SharedVolume: true,
			Nested:       false,
		},
		SharedVolume: SharedVolumeConfig{
			Paths: []string{},
		},
		Connect: ConnectConfig{
			Command: "/bin/sh",
		},
		Create: CreateConfig{
			ExtraArgs: []string{},
		},
	}, nil
}

// generateID returns an 8-character random lowercase alphanumeric string.
func generateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[n.Int64()]
	}
	return string(b)
}

// parseTOML decodes a silo TOML config file.
func parseTOML(path string) (Config, error) {
	var c Config
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// requireWorkspaceConfig returns the workspace config, or an error if no .silo/silo.toml exists.
func requireWorkspaceConfig() (Config, error) {
	if _, err := os.Stat(siloToml); os.IsNotExist(err) {
		return Config{}, fmt.Errorf("no .silo/silo.toml found — run silo first to initialize")
	}
	cfg, err := parseTOML(siloToml)
	if err != nil {
		return Config{}, fmt.Errorf("read silo.toml: %w", err)
	}
	return cfg, nil
}

// saveWorkspaceConfig writes the config to .silo/silo.toml, creating the directory as needed.
func (c Config) saveWorkspaceConfig() error {
	if err := os.MkdirAll(siloDir, 0755); err != nil {
		return err
	}
	f, err := os.Create(siloToml)
	if err != nil {
		return err
	}
	defer f.Close()
	if c.SharedVolume.Paths == nil {
		c.SharedVolume.Paths = []string{}
	}
	if c.Create.ExtraArgs == nil {
		c.Create.ExtraArgs = []string{}
	}
	return toml.NewEncoder(f).Encode(c)
}

// globalConfigDir returns $XDG_CONFIG_HOME/silo (defaults to ~/.config/silo).
func globalConfigDir() (string, error) {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "silo"), nil
}

// ensureFile creates the file at path with content if the file does not already exist.
// The parent directory is created with mode 0755 if needed.
func ensureFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.WriteFile(path, content, 0644)
	}
	return nil
}

// ensureGlobalHomeNix creates $XDG_CONFIG_HOME/silo/home.nix if the file does not already exist.
func ensureGlobalHomeNix() error {
	dir, err := globalConfigDir()
	if err != nil {
		return err
	}
	return ensureFile(filepath.Join(dir, "home.nix"), []byte(emptyHomeNix))
}

// ensureWorkspaceHomeNix creates .silo/home.nix if the file does not already exist.
func ensureWorkspaceHomeNix() error {
	return ensureFile(filepath.Join(siloDir, "home.nix"), []byte(emptyHomeNix))
}

// ensureGlobalDevcontainerJSON creates $XDG_CONFIG_HOME/silo/devcontainer.json
// with an empty JSON object if the file does not already exist.
func ensureGlobalDevcontainerJSON() error {
	dir, err := globalConfigDir()
	if err != nil {
		return err
	}
	return ensureFile(filepath.Join(dir, "devcontainer.json"), []byte("{}\n"))
}

// ensureGlobalConfig creates $XDG_CONFIG_HOME/silo/silo.toml as an empty file
// if the file does not already exist.
func ensureGlobalConfig() error {
	dir, err := globalConfigDir()
	if err != nil {
		return err
	}
	return ensureFile(filepath.Join(dir, "silo.toml"), []byte{})
}

// loadGlobalConfig parses $XDG_CONFIG_HOME/silo/silo.toml.
// The [general] section is not meaningful and is ignored.
// Returns an empty Config if the file does not exist.
func loadGlobalConfig() (Config, error) {
	dir, err := globalConfigDir()
	if err != nil {
		return Config{}, err
	}
	path := filepath.Join(dir, "silo.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Config{}, nil
	}
	return parseTOML(path)
}

// baseImageName returns the base image tag for the given user.
func baseImageName(user string) string {
	return "silo-" + user
}

// initWorkspaceConfig loads the workspace config, or seeds it from global defaults on first run.
func initWorkspaceConfig() (Config, error) {
	var cfg Config
	if _, statErr := os.Stat(siloToml); os.IsNotExist(statErr) {
		// First run: seed from global config, fall back to built-in defaults.
		defaults, err := defaultConfig()
		if err != nil {
			return cfg, err
		}
		if err := ensureGlobalConfig(); err != nil {
			return cfg, fmt.Errorf("init global silo.toml: %w", err)
		}
		cfg, err = loadGlobalConfig()
		if err != nil {
			return cfg, fmt.Errorf("read global silo.toml: %w", err)
		}
		cfg.General = defaults.General
		if cfg.Connect.Command == "" {
			cfg.Connect.Command = defaults.Connect.Command
		}
		if cfg.Features == (FeaturesConfig{}) {
			cfg.Features = defaults.Features
		}
		if err := cfg.saveWorkspaceConfig(); err != nil {
			return cfg, fmt.Errorf("save silo.toml: %w", err)
		}
	} else {
		// Subsequent runs: use workspace config as-is.
		var err error
		cfg, err = parseTOML(siloToml)
		if err != nil {
			return cfg, fmt.Errorf("read silo.toml: %w", err)
		}
	}
	return cfg, nil
}

// ensureScaffoldFiles ensures home.nix and devcontainer.json scaffold files exist.
func ensureScaffoldFiles() error {
	if err := ensureGlobalHomeNix(); err != nil {
		return err
	}
	if err := ensureWorkspaceHomeNix(); err != nil {
		return err
	}
	return ensureGlobalDevcontainerJSON()
}

// ensureImages builds the base and workspace images if they don't yet exist.
func ensureImages(cfg Config) error {
	tc := newTemplateContext(cfg)
	base := tc.BaseImage
	if !imageExists(base) {
		fmt.Printf("Building base image %s...\n", base)
		if err := buildBaseImage(base, tc); err != nil {
			return fmt.Errorf("build base image: %w", err)
		}
	}
	if !imageExists(cfg.General.ImageName) {
		fmt.Printf("Building workspace image %s...\n", cfg.General.ImageName)
		if err := buildWorkspaceImage(cfg.General.ImageName, tc); err != nil {
			return fmt.Errorf("build workspace image: %w", err)
		}
	}
	return nil
}
