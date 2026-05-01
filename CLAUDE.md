# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Silo

Silo is a Go CLI tool for creating per-directory developer containers powered by Podman, Nix, and home-manager. It provides isolated development environments with persistent shared storage across workspaces.

Requires Go 1.23+ and Podman.

## Build and Test Commands

```bash
go build .              # Build binary
go install .            # Install to $GOPATH/bin
go test ./...           # Run all tests
go test ./internal      # Run internal package tests
go test ./cmd           # Run cmd package tests
go test ./features      # Run feature spec tests
go test -run TestName  # Run a single test
go vet ./...            # Run static analysis
```

## Architecture

**Package structure:**
- `main.go` — CLI entry point and command dispatcher (package main)
- `cmd/` — Command implementations (package cmd)
- `internal/` — Helpers, config, container operations (package internal)
- `features/` — BDD-style feature spec tests (package features_test)

**Key files in `cmd/`:**
- `init.go` — `Init`, `UserInit`, `VolumeSetup`, `ParseInitFlags`
- `run.go` — `Run`, `Connect`, `Exec`, `ParseRunFlags`
- `stop.go` — `Stop`, `Status`, `Remove`, `UserRm`, `ParseRemoveFlags`, `parseForceFlag`
- `build.go` — `Build`, `UserBuild`
- `devcontainer.go` — `DevcontainerGenerate`, `DevcontainerStop`, `DevcontainerStatus`
- `create.go` — `Start` (create step is handled internally by `EnsureCreated`)

**Key files in `internal/`:**
- `config.go` — Config types, `ParseTOML`, `EnsureInit`, `EnsureBuilt`, etc.
- `container.go` — Container operations, `execCommand` seam
- `build.go` — `DetectNixSystem`, `ImageExists`, `BuildUserImage`, `BuildWorkspaceImage`
- `devcontainer.go` — `DevcontainerGenerate`, `DeepMergeJSON`
- `render.go` — Template rendering, `TemplateContext`, `HomeUserNix`
- `testutil.go` — Test helpers: `MockExecCommand`, `SetupWorkspace`, `MinimalConfig`

**Lifecycle chain** (each step depends on the ones before it):
```
init → build → create → start → connect
```

Note: `start` internally calls `EnsureCreated` (which creates the container if needed) and `VolumeSetup` before starting the container. `silo stop` stops and removes the container. `silo rm` removes the image.

**The `Ensure*` chain** provides lazy initialization:
- `EnsureInit` → initializes config and creates starter files
- `EnsureBuilt` → ensures images exist (builds if missing)
- `EnsureCreated` → ensures container exists (creates if missing)
- `EnsureStarted` → ensures container is running, calls `VolumeSetup` to create directories on the shared volume

**Configuration hierarchy** (later overrides earlier):
1. Built-in defaults
2. User config at `$XDG_CONFIG_HOME/silo/silo.in.toml`
3. Workspace config at `.silo/silo.toml`
4. Runtime flags

**Templates** in `templates/` are rendered using `text/template`. Path resolution uses `runtime.Caller(0)` to find the module root for both development and test execution.

**Two-stage image build:**
1. User image (`silo-<user>`): Alpine + Nix + home-manager, shared across workspaces
2. Workspace image (`silo-<id>`): Layered on user image with workspace-specific `home.nix`

**Shared volume:** The `silo-shared` named volume is mounted as subpath volumes at container paths (e.g., `/home/<user>/.cache/uv`). Paths in `[shared_volume]` are created on the volume before container start via `VolumeSetup`.

**Devcontainer merge:** `silo devcontainer` recursively merges `$XDG_CONFIG_HOME/silo/devcontainer.in.json` into generated `.devcontainer.json`.

**Only external dependency:** `github.com/BurntSushi/toml`

## TOML Style

Match the formatting in `examples/silo.in.toml`:
- Unindented keys (no leading spaces)
- 2-space array elements
- Blank line between tables

## Testing

Tests use a mock runner (`internal/testutil.go`) to stub Podman commands. The `var execCommand = exec.Command` seam in `internal/container.go` is swapped by `MockExecCommand` in tests, which records calls and returns preset responses keyed by the full command string.

**Do not use `t.Parallel()`** — tests call `os.Chdir` to a temp workspace, which is process-global. Use `SetupWorkspace` for chdir and cleanup.

Key test helpers in `internal/testutil.go`:
- `MockExecCommand(t, responses)` — installs mock, returns `*[]TestCmdCall` recorder
- `SetupWorkspace(t, cfg)` — creates temp dir with `.silo/silo.toml`, chdirs into it
- `SetupUserConfig(t)` — sets `XDG_CONFIG_HOME` to temp dir with starter files
- `MinimalConfig(id)` — returns a test-ready `Config` struct

Feature spec tests in `features/` use BDD-style `t.Run` nesting.

Check existing patterns in `*_test.go` before adding new tests.
