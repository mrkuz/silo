package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

// templatesPath returns the absolute path to the templates directory.
// For development (source tree), templates are at the module root.
// For installed binaries, templates are at PREFIX/share/silo/templates.
func templatesPath() string {
	// Use runtime.Caller to find source file location, then walk up to module root.
	// This works for both development and test execution.
	_, sourceFile, _, ok := runtime.Caller(0)
	if ok {
		moduleRoot := filepath.Dir(filepath.Dir(sourceFile))
		templates := filepath.Join(moduleRoot, "templates")
		if _, err := os.Stat(templates); err == nil {
			return templates
		}
	}

	// Fall back: walk up from CWD looking for go.mod and templates/
	dir, err := os.Getwd()
	if err == nil {
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				templates := filepath.Join(dir, "templates")
				if _, err := os.Stat(templates); err == nil {
					return templates
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Last resort: use executable location
	exe, err := os.Executable()
	if err != nil {
		return "/usr/local/share/silo/templates"
	}
	return filepath.Join(filepath.Dir(exe), "..", "share", "silo", "templates")
}

// ReadTemplate reads a template file from the templates directory.
func ReadTemplate(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(templatesPath(), name))
}

// HomeUserNix is the home-manager module for user configuration.
const HomeUserNix = `{
  config,
  pkgs,
  ...
}:
{
}
`

// WorkspaceHomeNixTmpl is the home-manager module for workspaces.
// It is rendered by RenderWorkspaceHomeNix with the podman parameter.
const WorkspaceHomeNixTmpl = `{
  config,
  pkgs,
  ...
}:
{
  silo.podman.enable = {{.Podman}};
}
`

// RenderWorkspaceHomeNix renders the workspace home.nix template.
func RenderWorkspaceHomeNix(podman bool) (string, error) {
	tmpl, err := template.New("home.nix").Parse(WorkspaceHomeNixTmpl)
	if err != nil {
		return "", fmt.Errorf("parse workspace home.nix template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct {
		Podman bool
	}{Podman: podman}); err != nil {
		return "", fmt.Errorf("render workspace home.nix template: %w", err)
	}
	return buf.String(), nil
}

// templateFuncs contains custom functions available in all templates.
var templateFuncs = template.FuncMap{
	"json": func(v any) (string, error) {
		b, err := json.Marshal(v)
		return string(b), err
	},
	"trimPrefix": strings.TrimPrefix,
}

// TemplateContext provides data for template rendering across devcontainer, Containerfile, and setup scripts.
type TemplateContext struct {
	User              string
	Home              string
	Image             string
	BaseImage         string
	ContainerName     string
	SharedVolumeName  string
	WorkspaceMount    string
	System            string
	ContainerArgs     []string
	DevcontainerArgs  []string
	SharedVolumePaths []string // resolved container paths for subpath mounts
}

// NewTemplateContext builds a TemplateContext from Config for template rendering.
// An optional suffix is appended to the container name.
func NewTemplateContext(cfg Config, containerNameSuffix ...string) (TemplateContext, error) {
	suffix := ""
	if len(containerNameSuffix) > 0 {
		suffix = containerNameSuffix[0]
	}
	containerName := ContainerNameWithSuffix(WorkspaceContainerName(cfg.General.ID), suffix)
	sharedVolumeNameValue := ""
	if cfg.Features.SharedVolume {
		sharedVolumeNameValue = cfg.GetSharedVolumeName()
	}

	home := "/home/" + cfg.General.User
	workspaceMount, err := WorkspaceMountPath(cfg)
	if err != nil {
		return TemplateContext{}, fmt.Errorf("resolve workspace mount path: %w", err)
	}
	// Build resolved container paths for each shared volume path
	var sharedPaths []string
	if cfg.Features.SharedVolume && len(cfg.SharedVolume.Paths) > 0 {
		sharedPaths = make([]string, len(cfg.SharedVolume.Paths))
		for i, path := range cfg.SharedVolume.Paths {
			sharedPaths[i] = ResolveContainerPath(path, cfg.General.User)
		}
	}
	// Build devcontainer args: name, hostname, and security args (no --user for devcontainer)
	devcontainerArgs := []string{"--name", containerName, "--hostname", containerName}
	if cfg.Features.Podman {
		devcontainerArgs = append(devcontainerArgs, "--security-opt", "label=disable", "--device", "/dev/fuse")
	} else {
		devcontainerArgs = append(devcontainerArgs, "--cap-drop=ALL", "--cap-add=NET_BIND_SERVICE", "--security-opt", "no-new-privileges")
	}

	return TemplateContext{
		User:              cfg.General.User,
		Home:              home,
		Image:             WorkspaceImageName(cfg.General.ID),
		BaseImage:         BaseImageName(cfg.General.User),
		ContainerName:     containerName,
		SharedVolumeName:  sharedVolumeNameValue,
		WorkspaceMount:    workspaceMount,
		System:            DetectNixSystem(),
		ContainerArgs:     ContainerArgs(cfg, containerNameSuffix...),
		DevcontainerArgs:  devcontainerArgs,
		SharedVolumePaths: sharedPaths,
	}, nil
}

// RenderTemplate parses and executes a template file with the given data.
func RenderTemplate(name string, data any) ([]byte, error) {
	content, err := ReadTemplate(name)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", name, err)
	}
	tmpl, err := template.New(name).Funcs(templateFuncs).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", name, err)
	}
	return buf.Bytes(), nil
}
