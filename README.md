# silo

Per-directory developer sandbox containers, powered by Podman, Nix, and home-manager.

---

## Features

- **Per-directory isolation** — each workspace gets its own container with a unique ID
- **Nix + home-manager** — global and per-workspace `home.nix` configs
- **Workspace mount** — the host directory is mounted inside the container automatically
- **Shared volume** — persist package caches and other data across containers and rebuilds
- **VS Code integration** — `silo devcontainer` generates a `.devcontainer.json`
- **Nested Podman** — optional support for running containers inside the container

---

## Quick Start

```bash
silo
```

See [Build and Install](#build-and-install) for installation instructions.

On first run, silo scaffolds `.silo/silo.toml` and builds two images: a shared base image and a per-workspace workspace image. Then it starts the container and connects to it.

See [Configuration](#configuration) to customize your workspace.

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

## Lifecycle

Every workspace goes through a fixed chain of steps. Each step depends on the ones before it. Running `silo` (or `silo connect`) triggers the full chain automatically.

```
init → build → create → start → setup → connect
```

| Step | Description | Output |
|---|---|---|
| **init** | Scaffolds `.silo/silo.toml` and `home.nix` on the host | Workspace config files |
| **build** | Builds the base image (shared) and the workspace image (per-workspace) | Container image |
| **create** | Creates the container from the workspace image | Stopped container |
| **start** | Starts the container | Running container |
| **setup** | Configure shared volume | Configured container |
| **connect** | Opens an interactive shell inside the running container | Terminal session |

---

## Commands

```
silo [--stop] [-- args...]
silo init
silo build [--base] [-f|--force]
silo create [--nested] [--no-workspace] [--no-shared-volume] [-f|--force] [--dry-run] [-- args...]
silo start [-f|--force]
silo setup
silo connect [--stop] [-- args...]
silo exec <cmd> [args...]
silo stop
silo rm [-f|--force] [--image]
silo status
silo devcontainer
silo help
```

### `silo connect` / `silo` (default)

Connect to the container for the current workspace. Runs the full lifecycle chain if needed. The bare `silo` invocation is an alias for `silo connect`.

| Flag | Description |
|---|---|
| `--stop` | Stop the container when the session exits |
| `-- ...` | Pass remaining arguments to `podman exec` |

### `silo init`

Initialize workspace and global config files. Creates `.silo/silo.toml`, `.silo/home.nix`, and global scaffold files on the host. Safe to run multiple times — existing files are not overwritten.

### `silo build`

Build the workspace image (and optionally the base image).

| Flag | Description |
|---|---|
| `--base` | Build the base image, then the workspace image |
| `-f`, `--force` | Remove and rebuild the image(s) if already present |

### `silo create`

Create the container without starting it. Builds images if needed.

| Flag | Description |
|---|---|
| `--nested` | Enable nested Podman containers (relaxes security opts, adds `/dev/fuse`) |
| `--no-workspace` | Disable workspace volume mount |
| `--no-shared-volume` | Disable shared volume |
| `-f`, `--force` | Remove and recreate the container if it already exists |
| `--dry-run` | Print the `podman create` command without running it |
| `-- ...` | Pass remaining arguments to `podman create` |

Feature flags (`--nested`, `--no-workspace`, `--no-shared-volume`) and extra arguments are persisted to `.silo/silo.toml`.

### `silo start`

Start the container and run post-start setup. Does not connect.

| Flag | Description |
|---|---|
| `-f`, `--force` | Restart the container if it is already running |

### `silo setup`

Run post-start setup inside the running container. Creates shared volume symlinks for paths configured in `[shared_volume]`. This step runs automatically after every start. Fails if the container is not running.

### `silo exec <cmd> [args...]`

Run a command inside the running container. Fails if the container is not running.

### `silo stop`

Stop the running container (immediate, no grace period).

### `silo rm`

Remove the container. Fails if the container is running unless `-f`/`--force` is given.

| Flag | Description |
|---|---|
| `-f`, `--force` | Stop the container if it is running before removing |
| `--image` | Also remove the workspace image |

### `silo status`

Print `Running` or `Stopped`.

### `silo devcontainer`

Generate a `.devcontainer.json` for VS Code in the current host directory. Does nothing if `.devcontainer.json` already exists. See [VS Code devcontainer](#vs-code-devcontainer) for details.

---

## Configuration

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
workspace     = true                 # Mount host directory into container
shared_volume = true                 # Mount shared volume
nested        = false                # Allow nested Podman containers

[shared_volume]
paths = [
    "$HOME/.local/share/fish/fish_history",  # persist and share file
    "$HOME/.cache/uv/",                      # persist and share directory (trailing /)
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
| `workspace` | `true` | Mount the host directory into the container at `/workspace/<id>/<dirname>` |
| `shared_volume` | `true` | Mount the `silo-shared` Podman volume at `/silo/shared` inside the container |
| `nested` | `false` | Enable nested Podman (adds `--device /dev/fuse`, disables SELinux label) |

**`[shared_volume]`**

| Key | Description |
|---|---|
| `paths` | Paths inside the container to back with the shared volume. See [Shared volume](#shared-volume). |

**`[create]`**

| Key | Default | Description |
|---|---|---|
| `extra_args` | `[]` | Extra arguments appended to `podman create`. |

`extra_args` is updated automatically when extra arguments are passed to `silo create`.

### Workspace config: `.silo/home.nix`

Home-manager config applied only to this workspace's image. Created as an empty module on first run.

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

### Global config: `$XDG_CONFIG_HOME/silo/`

On macOS the XDG spec is not followed by default, so `~/.config/silo/` is used unless `$XDG_CONFIG_HOME` is set explicitly.

All three files are created automatically on first run if they don't exist. See [examples/](examples/) for reference configs.

| File | Description |
|---|---|
| `silo.toml` | Default values for new workspaces. `[general]` is ignored. |
| `home.nix` | Global home-manager config baked into the base image. |
| `devcontainer.json` | Merged into every generated `.devcontainer.json`. |

---

## How It Works

### Two-stage image build

silo builds two OCI images using Podman:

1. **Base image** (`silo-<user>`) — shared across all workspaces. Alpine with Nix and home-manager installed. The global `home.nix` is baked in.
2. **Workspace image** (`silo-<id>`) — per-workspace, layered on top of the base. The workspace `home.nix` is applied here.

Build context files are written to a temporary directory on the host and passed to `podman build`. No persistent build context is kept on disk.

### Workspace mount

The host directory is mounted into the container at `/workspace/<id>/<dirname>`, where `<id>` is the workspace ID and `<dirname>` is the host directory's basename.

### Shared volume

The named Podman volume `silo-shared` is mounted at `/silo/shared` inside every container. Data stored there — such as package caches — is shared across all workspaces and survives container restarts and image rebuilds.

For paths listed in `[shared_volume]`, a symlink inside the container pointing to `/silo/shared/` is created on every container start. `$HOME` is the only supported placeholder. A trailing slash marks a directory; no trailing slash marks a file.

Example: `$HOME/.cache/uv/` creates a symlink from `$HOME/.cache/uv` to `/silo/shared/home/alice/.cache/uv` inside the container.

If a real file or directory already exists at the target path, the symlink is skipped and a warning is printed.

### Nix + home-manager

Each image build generates a Nix flake in a temporary directory on the host and passes it to `podman build`. The flake wires together `nixos-unstable`, home-manager, and `home.nix` (global and workspace).

### VS Code devcontainer

`silo devcontainer` generates a `.devcontainer.json` on the host, pointing at the workspace image. The generated name is `<container-name>-dev`. The global `$XDG_CONFIG_HOME/silo/devcontainer.json` is merged with the generated file — objects merge recursively, arrays concatenate.

**Important**

- The `silo` container is independent from the devcontainer.
- Lifecycle is managed by VS Code/devcontainers, not by `silo`.
- Shared volume is not supported
- `silo` commands (`start`/`stop`/`status`/`connect`/`rm`) target the regular workspace container, not the devcontainer.

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
