package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates
var templateFiles embed.FS

// homeUserNix is the home-manager module for user configuration.
const homeUserNix = `{
  config,
  pkgs,
  ...
}:
{
}
`

// emptyHomeNix is the default empty home-manager module for user images.
const emptyHomeNix = `{
  config,
  pkgs,
  ...
}:
{
}
`

// workspaceHomeNixTmpl is the home-manager module for workspaces.
// It is rendered by renderWorkspaceHomeNix with the podman parameter.
const workspaceHomeNixTmpl = `{
  config,
  pkgs,
  ...
}:
{
  module.podman.enable = {{.Podman}};
}
`

// renderWorkspaceHomeNix renders the workspace home.nix template.
func renderWorkspaceHomeNix(podman bool) (string, error) {
	tmpl, err := template.New("home.nix").Parse(workspaceHomeNixTmpl)
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
	SharedVolumePaths []string // resolved container paths for subpath mounts
}

// newTemplateContext builds a TemplateContext from Config for template rendering.
// An optional suffix is appended to the container name.
func newTemplateContext(cfg Config, containerNameSuffix ...string) (TemplateContext, error) {
	suffix := ""
	if len(containerNameSuffix) > 0 {
		suffix = containerNameSuffix[0]
	}
	containerName := containerNameWithSuffix(cfg.General.ContainerName, suffix)
	sharedVolumeNameValue := ""
	if cfg.Features.SharedVolume {
		sharedVolumeNameValue = cfg.getSharedVolumeName()
	}

	home := "/home/" + cfg.General.User
	workspaceMount, err := workspaceMountPath(cfg)
	if err != nil {
		return TemplateContext{}, fmt.Errorf("resolve workspace mount path: %w", err)
	}
	// Build resolved container paths for each shared volume path
	var sharedPaths []string
	if cfg.Features.SharedVolume && len(cfg.SharedVolume.Paths) > 0 {
		sharedPaths = make([]string, len(cfg.SharedVolume.Paths))
		for i, path := range cfg.SharedVolume.Paths {
			sharedPaths[i] = resolveContainerPath(path, cfg.General.User)
		}
	}
	return TemplateContext{
		User:              cfg.General.User,
		Home:              home,
		Image:             cfg.General.ImageName,
		BaseImage:         baseImageName(cfg.General.User),
		ContainerName:     containerName,
		SharedVolumeName:  sharedVolumeNameValue,
		WorkspaceMount:    workspaceMount,
		System:            detectNixSystem(),
		ContainerArgs:     containerArgs(cfg, containerNameSuffix...),
		SharedVolumePaths: sharedPaths,
	}, nil
}

// renderTemplate parses and executes an embedded Go template with the given data.
func renderTemplate(name string, data any) ([]byte, error) {
	content, err := templateFiles.ReadFile("templates/" + name)
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
