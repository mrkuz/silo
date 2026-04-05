package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"text/template"
)

//go:embed templates
var templateFiles embed.FS

// emptyHomeNix is the default empty home-manager module used as a workspace overlay.
const emptyHomeNix = `{
  config,
  pkgs,
  ...
}:
{
}
`

// templateFuncs contains custom functions available in all templates.
var templateFuncs = template.FuncMap{
	"json": func(v any) (string, error) {
		b, err := json.Marshal(v)
		return string(b), err
	},
}

// TemplateContext is the unified data object passed to every template.
type TemplateContext struct {
	Config           Config
	User             string
	Home             string
	Image            string
	BaseImage        string
	System           string
	ContainerArgs     []string
	SharedPathEntries []sharedPathEntry
}

// newTemplateContext builds a TemplateContext from the given Config.
func newTemplateContext(cfg Config) TemplateContext {
	home := "/home/" + cfg.General.User
	var entries []sharedPathEntry
	if cfg.Features.SharedVolume && len(cfg.SharedVolume.Paths) > 0 {
		entries = buildSharedVolumeEntries(cfg.SharedVolume.Paths)
	} else {
		entries = []sharedPathEntry{}
	}
	return TemplateContext{
		Config:            cfg,
		User:              cfg.General.User,
		Home:              home,
		Image:             cfg.General.ImageName,
		BaseImage:         baseImageName(cfg.General.User),
		System:            detectNixSystem(),
		ContainerArgs:      append(containerArgs(cfg.Features.Nested), "--hostname", cfg.General.ContainerName),
		SharedPathEntries: entries,
	}
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
