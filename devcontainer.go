package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const devContainerSuffix = "-dev"

// devContainerName returns the devcontainer name for the given config.
func devContainerName(cfg Config) string {
	return containerNameWithSuffix(cfg.General.ContainerName, devContainerSuffix)
}

// cmdDevcontainerGenerate generates a .devcontainer.json for VS Code.
func cmdDevcontainerGenerate() error {
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}

	tc, err := newTemplateContext(cfg, devContainerSuffix)
	if err != nil {
		return fmt.Errorf("build template context: %w", err)
	}
	content, err := renderTemplate("devcontainer.json.tmpl", tc)
	if err != nil {
		return fmt.Errorf("render devcontainer.json template: %w", err)
	}

	const devcontainerFile = ".devcontainer.json"

	if _, statErr := os.Stat(devcontainerFile); statErr == nil {
		return nil
	}

	userDC, err := loadDevcontainerInJSON()
	if err != nil {
		return fmt.Errorf("load devcontainer input file: %w", err)
	}
	if len(userDC) > 0 {
		var generated map[string]any
		if err := json.Unmarshal(content, &generated); err != nil {
			return fmt.Errorf("parse generated devcontainer.json: %w", err)
		}
		merged := deepMergeJSON(userDC, generated)
		content, err = json.MarshalIndent(merged, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal devcontainer.json: %w", err)
		}
		content = append(content, '\n')
	}
	if err := os.WriteFile(devcontainerFile, content, 0644); err != nil {
		return fmt.Errorf("write devcontainer.json: %w", err)
	}
	fmt.Printf("Generated %s.\n", devcontainerFile)
	return nil
}

// cmdDevcontainerStop implements `silo devcontainer stop`.
func cmdDevcontainerStop() error {
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	name := devContainerName(cfg)
	if !containerRunning(name) {
		fmt.Printf("%s is not running\n", name)
		return nil
	}
	if err := stopContainer(name); err != nil {
		return fmt.Errorf("stop container: %w", err)
	}
	return nil
}

// cmdDevcontainerStatus implements `silo devcontainer status`.
func cmdDevcontainerStatus() error {
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	name := devContainerName(cfg)
	if containerRunning(name) {
		fmt.Println("Running")
	} else {
		fmt.Println("Stopped")
	}
	return nil
}

// cmdDevcontainerRemove implements `silo devcontainer rm [-f|--force]`.
func cmdDevcontainerRemove(args []string) error {
	flags, err := parseDevcontainerRemoveFlags(args)
	if err != nil {
		return fmt.Errorf("parse devcontainer rm flags: %w", err)
	}
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("load workspace configuration: %w", err)
	}
	name := devContainerName(cfg)
	if containerExists(name) {
		if containerRunning(name) {
			if !flags.force {
				return fmt.Errorf("%s is running", name)
			}
			if err := stopContainer(name); err != nil {
				return fmt.Errorf("stop container before removal: %w", err)
			}
		}
		fmt.Printf("Removing %s...\n", name)
		if err := removeContainer(name); err != nil {
			return fmt.Errorf("remove container: %w", err)
		}
	} else {
		fmt.Printf("%s not found\n", name)
	}
	return nil
}

// loadDevcontainerInJSON reads the user devcontainer input file.
// Returns an empty map if the file does not exist.
func loadDevcontainerInJSON() (map[string]any, error) {
	dir, err := userConfigDir()
	if err != nil {
		return nil, fmt.Errorf("get user config directory: %w", err)
	}
	path := filepath.Join(dir, "devcontainer.in.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
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

// deepMergeJSON recursively merges input into base.
// Objects merge key-by-key, arrays concatenate, scalars from input override.
func deepMergeJSON(base, input map[string]any) map[string]any {
	result := make(map[string]any, len(base))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range input {
		if om, ok := v.(map[string]any); ok {
			if bm, ok := result[k].(map[string]any); ok {
				result[k] = deepMergeJSON(bm, om)
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
