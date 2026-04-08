# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Silo

Silo is a Go CLI tool for creating per-directory developer sandbox containers powered by Podman, Nix, and home-manager. It provides isolated development environments with persistent shared storage across workspaces.

## Build and Test Commands

```bash
go build .              # Build binary
go install .            # Install to $GOPATH/bin
go test ./...           # Run all tests
go test -run TestName   # Run a single test
```

Requires Go 1.23+ and Podman.

## Architecture

**Single-package design** — all source lives in `package main` with no internal packages.

**Key files:**
- `main.go` — CLI entry point and command dispatcher
- `config.go` — Configuration management (TOML-based, two-tier: user + workspace) and workspace initialization (`initWorkspaceConfig`)
- `container.go` — Podman container lifecycle (create, start, stop, exec, connect, rm, status)
- `build.go` — Two-stage image build (base image + workspace image)
- `devcontainer.go` — VS Code devcontainer.json generation with recursive JSON merge
- `render.go` — Embedded Go template rendering (`//go:embed templates/`)

**Two-stage image build:**
1. Base image (`silo-<user>`): Fedora + Nix + home-manager
2. Workspace image (`silo-<id>`): Layered on base with workspace-specific Nix packages

**Configuration hierarchy** (later overrides earlier):
1. Built-in defaults
2. User config at `$XDG_CONFIG_HOME/silo/`
3. Workspace config at `.silo/silo.toml`
4. Runtime flags

**Templates** in `templates/` are embedded into the binary via `//go:embed` and rendered with `text/template`.

**Shared volume mounts:** The `silo-shared` named volume is mounted at `/shared` as a single bind. Paths in `[shared_volume]` become symlinks from the target path to a mirrored location under `/shared` (e.g. `$HOME/.cache/uv/` → `/shared/home/user/.cache/uv/`). The symlink script runs via `podman exec` after the container starts. A trailing slash means directory; no slash means file.

**Devcontainer merge:** `silo devcontainer` merges `$XDG_CONFIG_HOME/silo/devcontainer.in.json` into the generated output — objects merge recursively (key-by-key), arrays concatenate (base first), scalars from input win.

**Only external dependency:** `github.com/BurntSushi/toml`

## Testing

Tests use a mock runner (`mock_test.go`) to stub out Podman commands. The `var execCommand = exec.Command` seam in `container.go` is swapped by `mockExecCommand` in tests, which records all calls and returns preset responses keyed by the full command string.

**Do not use `t.Parallel()`** — tests call `os.Chdir` to a temp workspace, which is process-global. The `setupWorkspace` helper handles chdir and cleanup.

Key test helpers in `mock_test.go`:
- `mockExecCommand(t, responses)` — installs the mock, returns `*[]cmdCall` recorder
- `setupWorkspace(t, cfg)` — creates temp dir with `.silo/silo.toml`, chdirs into it
- `setupUserConfig(t)` — sets `XDG_CONFIG_HOME` to a temp dir with starter files (`home-user.nix`, `silo.in.toml`, etc.)
- `minimalConfig(id)` — returns a test-ready `Config` struct

Check existing patterns in `*_test.go` files before adding new tests.
