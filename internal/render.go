package internal

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates/*
var templatesFS embed.FS

// ReadTemplate reads a template file from the embedded templates.
func ReadTemplate(name string) ([]byte, error) {
	return templatesFS.ReadFile("templates/" + name)
}

// DetectNixSystem returns the Nix system identifier for the current machine architecture.
func DetectNixSystem() string {
	out, err := ExecCommand("uname", "-m").Output()
	if err != nil {
		return "x86_64-linux"
	}

	switch strings.TrimSpace(string(out)) {
	case "aarch64", "arm64":
		return "aarch64-linux"
	default:
		return "x86_64-linux"
	}
}

const EmptyDevcontainerUserJSON = "{}\n"

// SiloDevcontainerJSON is the base .devcontainer/devcontainer.json content.
const SiloDevcontainerJSON = `{
  "customizations": {
    "vscode": {
      "extensions": []
    }
  }
}
`

// EmptyHomeNix is the empty home-manager module used in the build context.
const EmptyHomeNix = `{ config, pkgs, ... }:
{
}
`

// HomeUserNix is the home-manager module for user configuration.
const HomeUserNix = `{ config, pkgs, ... }:
{
  silo.shellCommand = "${pkgs.bash}/bin/bash --login";
}
`

// WorkspaceHomeNixTmpl is the home-manager module for workspaces.
// It is rendered by RenderWorkspaceHomeNix with the podman parameter.
const WorkspaceHomeNixTmpl = `{ config, pkgs, ... }:
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
	User                    string
	Home                    string
	Image                   string
	ContainerName           string
	PersistenceVolumeName   string
	WorkspaceMount          string
	SiloID                  string
	System                  string
	DevcontainerArgs        []string
	PersistenceSharedPaths  []string
	PersistencePrivatePaths []string
	NetworkPorts            []string
}

// NewTemplateContext builds a TemplateContext from MergedConfig for template rendering.
// An optional suffix is appended to the container name.
func NewTemplateContext(cfg MergedConfig, containerNameSuffix ...string) (TemplateContext, error) {
	suffix := ""
	if len(containerNameSuffix) > 0 {
		suffix = containerNameSuffix[0]
	}
	containerName := containerNameWithSuffix(WorkspaceContainerName(cfg.ID), suffix)
	sharedVolumeNameValue := ""
	if len(cfg.Persistence.SharedPaths) > 0 || len(cfg.Persistence.PrivatePaths) > 0 {
		sharedVolumeNameValue = "silo"
	}

	home := "/home/" + cfg.User
	workspaceConfig := WorkspaceConfig{General: WorkspaceGeneralConfig{ID: cfg.ID}}
	workspaceMount, err := WorkspaceMountPath(workspaceConfig)
	if err != nil {
		return TemplateContext{}, fmt.Errorf("resolve workspace mount path: %w", err)
	}
	var sharedPaths []string
	if len(cfg.Persistence.SharedPaths) > 0 {
		sharedPaths = make([]string, len(cfg.Persistence.SharedPaths))
		for i, path := range cfg.Persistence.SharedPaths {
			sharedPaths[i] = ResolveContainerPath(path, cfg.User)
		}
	}
	var privatePaths []string
	if len(cfg.Persistence.PrivatePaths) > 0 {
		privatePaths = make([]string, len(cfg.Persistence.PrivatePaths))
		for i, path := range cfg.Persistence.PrivatePaths {
			privatePaths[i] = ResolveContainerPath(path, cfg.User)
		}
	}
	devcontainerArgs := append([]string{}, ContainerArgs(workspaceConfig, suffix)...)
	devcontainerArgs = append(devcontainerArgs, DefaultCreateArgs(cfg.Features.Podman)...)
	devcontainerArgs = append(devcontainerArgs, LimitsArgs(cfg.Limits)...)

	return TemplateContext{
		User:                    cfg.User,
		Home:                    home,
		Image:                   WorkspaceImageName(cfg.ID),
		ContainerName:           containerName,
		PersistenceVolumeName:   sharedVolumeNameValue,
		WorkspaceMount:          workspaceMount,
		SiloID:                  cfg.ID,
		System:                  DetectNixSystem(),
		DevcontainerArgs:        devcontainerArgs,
		PersistenceSharedPaths:  sharedPaths,
		PersistencePrivatePaths: privatePaths,
		NetworkPorts:            cfg.Network.Ports,
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
