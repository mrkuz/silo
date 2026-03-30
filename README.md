# silo

Per-directory developer sandbox containers, powered by Podman, Nix, and home-manager. Run `silo` in any directory to connect to the right container — images and containers are built and started automatically.

---

## Features

- **Per-directory isolation** — each directory gets its own container with a unique ID
- **Nix + home-manager** — global and per-workspace `home.nix` configs
- **Workspace mount** — current directory is mounted inside the container automatically
- **Shared volume** — persist package caches and other data across containers and rebuilds
- **VS Code integration** — `silo devcontainer` generates a `.devcontainer.json`
- **Nested Podman** — optional support for running containers inside the container

---

## Quick Start

```bash
silo
```

See [Build and Install](#build-and-install) for installation instructions.

On first run, silo scaffolds `.silo/silo.toml` and builds two images: a shared base image and a per-directory workspace image. Then it connects to the container. Subsequent runs reconnect to the running container or restart it if stopped.

---

## Build and Install

**Requirements:** Go 1.23+, Podman.

```bash
# Build binary
go build .

# Install to $GOPATH/bin
go install .
```

---

## Commands

```
silo [--stop] [-- args...]
silo connect [--stop] [-- args...]
silo start [--force]
silo create [--nested] [--no-workspace] [--no-shared-volume] [--force] [--dry-run] [-- args...]
silo build [--base] [--force]
silo exec <cmd> [args...]
silo stop
silo rm [--image]
silo status
silo devcontainer
silo help
```

### `silo connect` / `silo` (default)

Connect to the container for the current directory. Builds images and container on first run. The bare `silo` invocation is an alias for `silo connect`.

| Flag | Description |
|---|---|
| `--stop` | Stop the container when the session exits |
| `-- ...` | Pass remaining arguments to `podman exec` |

### `silo start`

Start the container without connecting to it.

| Flag | Description |
|---|---|
| `--force` | Restart the container if it is already running |

### `silo create`

Create the container without starting it. Builds images if needed.

| Flag | Description |
|---|---|
| `--nested` | Enable nested Podman containers (relaxes security opts, adds `/dev/fuse`) |
| `--no-workspace` | Disable workspace volume mount |
| `--no-shared-volume` | Disable shared volume |
| `--force` | Remove and recreate the container if it already exists |
| `--dry-run` | Print the `podman create` command without running it |
| `-- ...` | Pass remaining arguments to `podman create` |

### `silo build`

Build the workspace image (and optionally the base image).

| Flag | Description |
|---|---|
| `--base` | Build the base image, then the workspace image |
| `--force` | Remove and rebuild the image(s) if it already exists |

### `silo exec <cmd> [args...]`

Run a command in the running container. Fails if the container is not running.

### `silo stop`

Stop the running container (immediate, no grace period).

### `silo rm [--image]`

Remove the container. Pass `--image` to also remove the workspace image.

### `silo status`

Print `Running` or `Stopped`.

### `silo devcontainer`

Generate a `.devcontainer.json` for VS Code in the current directory. Merges `$XDG_CONFIG_HOME/silo/devcontainer.json` with silo-specific settings (image name, user, security options). Does nothing if `.devcontainer.json` already exists.

---

## Configuration

`$XDG_CONFIG_HOME` defaults to `~/.config`. On macOS the XDG spec is not followed by default, so `~/.config/silo/` is used unless `$XDG_CONFIG_HOME` is set explicitly.

### Workspace config: `.silo/silo.toml`

Created automatically on first run. Scaffolded from `$XDG_CONFIG_HOME/silo/silo.toml` if present.

```toml
[general]
id             = "ab3f9c12"          # 8-char random ID; names container and image
user           = "alice"
container_name = "silo-ab3f9c12"
image_name     = "silo-ab3f9c12"

[connect]
command = "/bin/sh"                  # Command executed when connecting to container

[features]
workspace     = true                 # Mount current directory into container
shared_volume = true                 # Mount shared Podman volume
nested        = false                # Allow nested Podman containers

[shared_volume]
paths = [
    "$HOME/.local/share/fish/fish_history",  # persisted file
    "$HOME/.cache/uv/",                      # persisted directory (trailing /)
]

[create]
extra_args = []                      # Extra arguments passed to podman create
```

**`[connect]`**

| Key | Default | Description |
|---|---|---|
| `command` | `/bin/sh` | Command executed when connecting to the container |

**`[features]`**

| Key | Default | Description |
|---|---|---|
| `workspace` | `true` | Mount `$PWD` into the container at `/workspace/<id>/<dirname>` |
| `shared_volume` | `true` | Mount the `silo-shared` Podman volume at `/shared` |
| `nested` | `false` | Enable nested Podman (adds `--device /dev/fuse`, disables SELinux label) |

**`[shared_volume]`**

| Key | Description |
|---|---|
| `paths` | Paths to persist via symlinks into `/shared`. `$HOME` is expanded at runtime. A trailing `/` marks a directory; without it, a file symlink is created. |

**`[create]`**

| Key | Default | Description |
|---|---|---|
| `extra_args` | `[]` | Extra arguments appended to `podman create`. |

`extra_args` is updated automatically when `-- ...` arguments are passed to `silo create`, so ad-hoc flags are persisted for future runs.

### Global config: `$XDG_CONFIG_HOME/silo/`

| File | Description |
|---|---|
| `silo.toml` | Default values for new workspaces. `[general]` is ignored. |
| `home.nix` | Global home-manager config applied to every container. |
| `devcontainer.json` | Merged into every generated `.devcontainer.json`. |

### Per-workspace Nix config: `.silo/home.nix`

Home-manager config applied only to this workspace. Created as an empty module on first run.

Example:

```nix
{ config, pkgs, ... }:
{
  home.packages = with pkgs; [
    nodejs
    python3
  ];
}
```

---

## How It Works

### Two-stage image build

silo builds two OCI images using Podman:

1. **Base image** (`silo-<user>`) — shared across all workspaces. Built on Alpine with Nix and home-manager installed.
2. **Workspace image** (`silo-<id>`) — per-directory, layered on top of the base.

Build context files are written to a temporary directory and passed to `podman build`. No persistent build context is kept on disk.

### Container lifecycle

On each `silo` invocation:

1. If `.silo/silo.toml` doesn't exist, a new config is scaffolded with a random 8-character ID.
2. Missing images are built automatically (base first, then workspace).
3. If the container is running: connect to it. If stopped: start it, then connect. If missing: create it, start it, run shared volume setup, then connect.

### Workspace mount

The current directory is mounted at `/workspace/<id>/<dirname>` inside the container.

### Shared volume

The named Podman volume `silo-shared` is mounted at `/shared` in every container, so data — such as package caches — is shared across all workspaces and survives container restarts and image rebuilds.

Paths listed in `[shared_volume] paths` are redirected via symlinks — the path inside the container looks normal, but the data lives in `/shared/`, mirroring the absolute path. `$HOME` is the only supported placeholder. For example, `$HOME/.cache/uv/` maps to `/shared/home/alice/.cache/uv/`.

### Nix + home-manager

Each image build generates a Nix flake in a temporary directory and passes it to `podman build`. The flake wires together `nixos-unstable` and home-manager, and loads two modules: `home.nix` (global, from `$XDG_CONFIG_HOME/silo/`) and `home.nix` (workspace, from `.silo/`). The global config is baked into the base image; the workspace config into the workspace image. The target architecture is detected via `uname -m`.

### VS Code devcontainer

`silo devcontainer` generates a `.devcontainer.json` pointing at the workspace image. The global `$XDG_CONFIG_HOME/silo/devcontainer.json` is deep-merged with the generated file — objects merge recursively, arrays concatenate. This lets you define shared extensions and settings once.

Example global `$XDG_CONFIG_HOME/silo/devcontainer.json`:

```json
{
  "customizations": {
    "vscode": {
      "extensions": [
        "lfs.vscode-emacs-friendly"
      ]
    }
  }
}
```
