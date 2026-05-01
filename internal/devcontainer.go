package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const devContainerSuffix = "-dev"
const devcontainerFileMode = 0644

// DevContainerName returns the devcontainer name for the given config.
func DevContainerName(cfg Config) string {
	return ContainerNameWithSuffix(cfg.General.ContainerName, devContainerSuffix)
}

// DevcontainerGenerate generates a .devcontainer.json for VS Code.
// If force is true, the file is always overwritten.
func DevcontainerGenerate(force bool) error {
	cfg, err := RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}

	// Ensure volume directories exist before generating devcontainer.json
	if cfg.Features.SharedVolume && len(cfg.SharedVolume.Paths) > 0 {
		if _, err := VolumeSetup(cfg); err != nil {
			return fmt.Errorf("volume setup: %w", err)
		}
	}

	tc, err := NewTemplateContext(cfg, devContainerSuffix)
	if err != nil {
		return fmt.Errorf("build template context: %w", err)
	}
	content, err := RenderTemplate("devcontainer.json.tmpl", tc)
	if err != nil {
		return fmt.Errorf("render devcontainer.json template: %w", err)
	}

	const devcontainerFile = ".devcontainer.json"

	if _, statErr := os.Stat(devcontainerFile); statErr == nil && !force {
		PrintInitFileStatus(devcontainerFile)
		return nil
	}

	userDC, err := LoadDevcontainerInJSON()
	if err != nil {
		return fmt.Errorf("load devcontainer input file: %w", err)
	}
	if len(userDC) > 0 {
		var generated map[string]any
		if err := json.Unmarshal(content, &generated); err != nil {
			return fmt.Errorf("parse generated devcontainer.json: %w", err)
		}
		merged := DeepMergeJSON(generated, userDC)
		content, err = json.MarshalIndent(merged, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal devcontainer.json: %w", err)
		}
		content = append(content, '\n')
	}
	if err := os.WriteFile(devcontainerFile, content, devcontainerFileMode); err != nil {
		return fmt.Errorf("write devcontainer.json: %w", err)
	}
	fmt.Printf("Generated %s\n", devcontainerFile)
	return nil
}

// DevcontainerStop implements `silo devcontainer stop`.
func DevcontainerStop() error {
	cfg, err := RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	name := DevContainerName(cfg)
	if !ContainerExists(name) {
		PrintNotFound(name)
		return nil
	}
	if ContainerRunning(name) {
		if err := StopContainer(name); err != nil {
			return fmt.Errorf("stop container: %w", err)
		}
	} else {
		fmt.Printf("%s is not running\n", name)
	}
	fmt.Printf("Removing %s...\n", name)
	if err := RemoveContainer(name); err != nil {
		return fmt.Errorf("remove container: %w", err)
	}
	return nil
}

// DevcontainerStatus implements `silo devcontainer status`.
func DevcontainerStatus() error {
	cfg, err := RequireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	name := DevContainerName(cfg)
	PrintRunningStatus(ContainerRunning(name))
	return nil
}

// LoadDevcontainerInJSON reads the user devcontainer input file.
// Returns an empty map if the file does not exist.
func LoadDevcontainerInJSON() (map[string]any, error) {
	dir, err := UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("get user config directory: %w", err)
	}
	path := filepath.Join(dir, "devcontainer.in.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read devcontainer input file: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse devcontainer input JSON: %w", err)
	}
	return m, nil
}

// DeepMergeJSON performs a deep merge of two maps.
func DeepMergeJSON(base, input map[string]any) map[string]any {
	result := make(map[string]any, len(base))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range input {
		if om, ok := v.(map[string]any); ok {
			if bm, ok := result[k].(map[string]any); ok {
				result[k] = DeepMergeJSON(bm, om)
				continue
			}
		}
		if oa, ok := v.([]any); ok {
			if ba, ok := result[k].([]any); ok {
				result[k] = append(ba, oa...)
				continue
			}
		}
		result[k] = v
	}
	return result
}
