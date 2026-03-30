package main

import (
	"bytes"
	"embed"
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

// renderTemplate parses and executes an embedded Go template with the given data.
func renderTemplate(name string, data any) ([]byte, error) {
	content, err := templateFiles.ReadFile("templates/" + name)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", name, err)
	}
	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", name, err)
	}
	return buf.Bytes(), nil
}
