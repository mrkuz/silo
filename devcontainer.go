package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// cmdDevcontainer implements `silo devcontainer`.
func cmdDevcontainer() error {
	cfg, err := requireWorkspaceConfig()
	if err != nil {
		return err
	}

	tc := newTemplateContext(cfg, "-dev")
	content, err := renderTemplate("devcontainer.json.tmpl", tc)
	if err != nil {
		return err
	}

	const devcontainerFile = ".devcontainer.json"

	if _, statErr := os.Stat(devcontainerFile); statErr == nil {
		return nil
	}

	globalDC, err := loadGlobalDevcontainerJSON()
	if err != nil {
		return fmt.Errorf("read global devcontainer.json: %w", err)
	}
	if len(globalDC) > 0 {
		var generated map[string]any
		if err := json.Unmarshal(content, &generated); err != nil {
			return fmt.Errorf("parse generated devcontainer.json: %w", err)
		}
		merged := deepMergeJSON(globalDC, generated)
		content, err = json.MarshalIndent(merged, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal devcontainer.json: %w", err)
		}
		content = append(content, '\n')
	}
	if err := os.WriteFile(devcontainerFile, content, 0644); err != nil {
		return err
	}
	fmt.Printf("Generated %s.\n", devcontainerFile)
	return nil
}

// loadGlobalDevcontainerJSON reads $XDG_CONFIG_HOME/silo/devcontainer.json.
// Returns an empty map if the file does not exist.
func loadGlobalDevcontainerJSON() (map[string]any, error) {
	dir, err := globalConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "devcontainer.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse global devcontainer.json: %w", err)
	}
	return m, nil
}

// deepMergeJSON recursively merges overlay into base and returns a new map.
// Objects are merged key-by-key. Arrays are concatenated (base first).
// Scalar values from overlay override base.
func deepMergeJSON(base, overlay map[string]any) map[string]any {
	result := make(map[string]any, len(base))
	for k, v := range base {
		result[k] = v
	}
	for k, ov := range overlay {
		if om, ok := ov.(map[string]any); ok {
			if bm, ok := result[k].(map[string]any); ok {
				result[k] = deepMergeJSON(bm, om)
				continue
			}
		}
		if oa, ok := ov.([]any); ok {
			if ba, ok := result[k].([]any); ok {
				result[k] = append(ba, oa...)
				continue
			}
		}
		result[k] = ov
	}
	return result
}
