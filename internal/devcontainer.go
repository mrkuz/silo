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

const siloDevcontainerBoilerplate = `{
  "customizations": {
    "vscode": {
      "extensions": []
    }
  }
}
`

// DevContainerName returns the devcontainer name for the given merged config.
func DevContainerName(cfg MergedConfig) string {
	return containerNameWithSuffix(WorkspaceContainerName(cfg.ID), devContainerSuffix)
}

// DevcontainerGenerate generates a .devcontainer.json for VS Code.
// If force is true, the file is always overwritten.
func DevcontainerGenerate(force bool) error {
	cfg, err := RequireMergedConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}

	// Ensure volume directories exist before generating devcontainer.json
	if len(cfg.Persistence.SharedPaths) > 0 {
		if _, err := VolumeSetup(cfg); err != nil {
			return fmt.Errorf("volume setup: %w", err)
		}
	}

	const siloDevcontainerFile = ".silo/devcontainer.json"
	if err := ensureSiloDevcontainerJSON(siloDevcontainerFile); err != nil {
		return fmt.Errorf("ensure project devcontainer json: %w", err)
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

	var fileExisted bool
	if _, statErr := os.Stat(devcontainerFile); statErr == nil {
		fileExisted = true
	}

	if _, statErr := os.Stat(devcontainerFile); statErr == nil && !force {
		PrintInitFileStatus(devcontainerFile)
		return nil
	}

	userDC, err := LoadDevcontainerInJSON()
	if err != nil {
		return fmt.Errorf("load devcontainer user file: %w", err)
	}

	siloDC, err := LoadSiloDevcontainerJSON()
	if err != nil {
		return fmt.Errorf("load silo devcontainer file: %w", err)
	}

	var generated map[string]any
	if err := json.Unmarshal(content, &generated); err != nil {
		return fmt.Errorf("parse generated devcontainer.json: %w", err)
	}

	merged := DeepMergeJSON(userDC, siloDC)
	merged = DeepMergeJSON(merged, generated)

	content, err = json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devcontainer.json: %w", err)
	}
	content = append(content, '\n')

	if fileExisted && force {
		fmt.Printf("'.devcontainer.json' updated\n")
	} else {
		PrintInitFileStatus(devcontainerFile)
	}
	if err := os.WriteFile(devcontainerFile, content, devcontainerFileMode); err != nil {
		return fmt.Errorf("write devcontainer.json: %w", err)
	}
	return nil
}

// LoadDevcontainerInJSON reads the user devcontainer input file.
// Returns an empty map if the file does not exist.
func LoadDevcontainerInJSON() (map[string]any, error) {
	dir, err := UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("get user config directory: %w", err)
	}
	path := filepath.Join(dir, "devcontainer.user.json")
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

// LoadSiloDevcontainerJSON reads the workspace .silo/devcontainer.json file.
// Returns an empty map if the file does not exist.
func LoadSiloDevcontainerJSON() (map[string]any, error) {
	path := ".silo/devcontainer.json"
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read silo devcontainer file: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse .silo/devcontainer.json: %w", err)
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

// ensureSiloDevcontainerJSON creates .silo/devcontainer.json with boilerplate content
// if it does not exist. Uses PrintInitFileStatus to print info messages.
func ensureSiloDevcontainerJSON(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create .silo directory: %w", err)
	}
	if _, statErr := os.Stat(path); statErr == nil {
		PrintInitFileStatus(path)
		return nil
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("stat %s: %w", path, statErr)
	}
	PrintInitFileStatus(path)
	if err := os.WriteFile(path, []byte(siloDevcontainerBoilerplate), devcontainerFileMode); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
