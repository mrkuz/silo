# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Silo

Silo is a Go CLI tool for creating per-directory developer sandbox containers powered by Podman, Nix, and home-manager. It provides isolated development environments with persistent shared storage across workspaces.

## Build and Test Commands

```bash
go build .             # Build binary
go install .           # Install to $GOPATH/bin
go test ./...          # Run all tests
go test -run TestName  # Run a single test
```

Requires Go 1.23+ and Podman.

## Architecture

**Single-package design** — all source lives in `package main` with no internal packages.

**Key files:**
- `main.go` — CLI entry point and command dispatcher
- `commands.go` — Command implementations (`cmdInit`, `cmdCreate`, etc.)
- `config.go` — TOML configuration management (user + workspace tier) and workspace initialization
- `container.go` — Podman container lifecycle and the `ensure*` chain
- `build.go` — Two-stage image build (user image + workspace image)
- `devcontainer.go` — VS Code devcontainer.json generation with recursive JSON merge
- `render.go` — Embedded Go template rendering (`//go:embed templates/`)

**Lifecycle chain** (each step depends on the ones before it):
```
init → build → create → start → connect
```

**`silo init` flags** use tri-state booleans (`--podman`/`--no-podman`, `--shared-volume`/`--no-shared-volume`). Flags not provided leave the config value unchanged; provided flags override the `silo.in.toml` default. Config is written only on first run.

**The `ensure*` chain** in `container.go` provides lazy initialization:
- `ensureInit` → initializes config and creates starter files
- `ensureBuilt` → ensures images exist (builds if missing)
- `ensureCreated` → ensures container exists (creates if missing)
- `ensureStarted` → ensures container is running

**Configuration hierarchy** (later overrides earlier):
1. Built-in defaults
2. User config at `$XDG_CONFIG_HOME/silo/silo.in.toml`
3. Workspace config at `.silo/silo.toml`
4. Runtime flags

**Templates** in `templates/` are embedded via `//go:embed` and rendered with `text/template`.

**Two-stage image build:**
1. User image (`silo-<user>`): Alpine + Nix + home-manager, shared across workspaces
2. Workspace image (`silo-<id>`): Layered on user image with workspace-specific `home.nix`

**Shared volume:** The `silo-shared` named volume is mounted as subpath volumes at container paths (e.g., `/home/<user>/.cache/uv`). Paths in `[shared_volume]` are created on the volume before container start via `ensureVolumeSetup`.

**Devcontainer merge:** `silo devcontainer` recursively merges `$XDG_CONFIG_HOME/silo/devcontainer.in.json` into generated `.devcontainer.json`.

**Only external dependency:** `github.com/BurntSushi/toml`

## TOML Style

Match the formatting in `examples/silo.in.toml`:
- Unindented keys (no leading spaces)
- 2-space array elements
- Blank line between tables

## Testing

Tests use a mock runner (`mock_test.go`) to stub Podman commands. The `var execCommand = exec.Command` seam in `container.go` is swapped by `mockExecCommand` in tests, which records calls and returns preset responses keyed by the full command string.

**Do not use `t.Parallel()`** — tests call `os.Chdir` to a temp workspace, which is process-global. Use `setupWorkspace` for chdir and cleanup.

Key test helpers in `mock_test.go`:
- `mockExecCommand(t, responses)` — installs mock, returns `*[]cmdCall` recorder
- `setupWorkspace(t, cfg)` — creates temp dir with `.silo/silo.toml`, chdirs into it
- `setupUserConfig(t)` — sets `XDG_CONFIG_HOME` to temp dir with starter files
- `minimalConfig(id)` — returns a test-ready `Config` struct

Check existing patterns in `*_test.go` before adding new tests.
