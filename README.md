# silo

Per-directory developer sandbox containers, powered by Podman, Nix, and home-manager.

---

## Features

- **Per-directory isolation** — each workspace gets its own container with a unique ID
- **Nix + home-manager** — shared `home-user.nix` and per-workspace `.silo/home.nix`
- **Workspace mount** — the host directory is mounted inside the container automatically
- **Shared volume** — persist package caches and other data across containers and rebuilds
- **VS Code integration** — `silo devcontainer` generates a `.devcontainer.json`
- **Nested Podman** — optional support for running containers inside the container

---

## Quick Start

```bash
silo
```

On first run, silo initializes workspace files, builds the user image and workspace image, starts the container, and connects to it. Subsequent runs skip steps that are already complete and connect directly.

See [Build and Install](#build-and-install) for installation instructions. See [Configuration](#configuration) to customize your workspace.

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

| Step | Description | Output | Idempotency |
|---|---|---|---|
| **init** | Creates `.silo/silo.toml`, `.silo/home.nix`; runs `silo user init` to create user files | Workspace + user files | Writes config only on first run |
| **build** | Ensures user image exists, then builds workspace image if needed | Container image | Images are cached; only missing ones are built |
| **create** | Creates the container from the workspace image | Stopped container | Skipped if container already exists |
| **start** | Starts the stopped container | Running container | Skipped if container is already running |
| **setup** | Runs shared volume symlink setup inside the running container | Configured container | Safe to re-run |
| **connect** | Opens an interactive shell inside the running container | Terminal session | — |

You can run individual steps:

```bash
silo init       # Initialize workspace and user files
silo build      # Build images (if not already built)
silo create     # Create container (if not already created)
silo start      # Start container (if not already running)
silo setup      # Run post-start setup
silo connect    # Connect to container (triggers missing steps automatically)
```

---

## Commands

```
silo [--stop|--rm|--rmi] [-- args...]
silo init [--nested|--no-nested] [--shared-volume|--no-shared-volume]
silo build
silo create [--dry-run] [-- args...]
silo start
silo setup
silo connect
silo exec <cmd> [args...]
silo stop
silo rm [-f|--force]
silo rmi [-f|--force]
silo status
silo user init
silo user build
silo user rmi
silo devcontainer
silo devcontainer stop
silo devcontainer rm [--force]
silo devcontainer status
silo help
```

### `silo` (default)

Run the full lifecycle chain if needed, then connect to the container for the current workspace.

| Flag | Description |
|---|---|
| `--stop` | Stop the container when the session exits |
| `--rm` | Stop and remove the container when the session exits |
| `--rmi` | Stop, remove container, and remove workspace image when the session exits |
| `-- ...` | Pass remaining arguments to `podman exec` |

### `silo init`

Initialize workspace files. Creates `.silo/silo.toml` and `.silo/home.nix`, then delegates to `silo user init` for user files. Writes config only on first run. If a flag is not provided, the default from `silo.in.toml` is used; if that's also unset, built-in defaults apply.

| Flag | Description |
|---|---|
| `--nested` | Enable nested Podman containers |
| `--no-nested` | Disable nested Podman containers |
| `--shared-volume` | Enable shared volume |
| `--no-shared-volume` | Disable shared volume |

### `silo build`

Ensure the user image exists, then build the workspace image if it does not exist yet.

### `silo create`

Create the container without starting it. Builds images if needed.

| Flag | Description |
|---|---|
| `--dry-run` | Print the `podman create` command without running it |
| `-- ...` | Pass remaining arguments to `podman create` |

`extra_args` in `.silo/silo.toml` is updated automatically when extra arguments are passed to `silo create`.

### `silo start`

Start the container and run post-start setup. Does not connect. If the container is already running, this command does nothing.

### `silo setup`

Run post-start setup inside the running container. Creates shared volume symlinks for paths configured in `[shared_volume]`. This step runs automatically after every start. Fails if the container is not running.

### `silo connect`

Connect to the container for the current workspace. Runs the full lifecycle chain automatically if any step has not completed yet, then opens an interactive shell inside the running container.

### `silo exec <cmd> [args...]`

Run a command inside the running container. Fails if the container is not running.

### `silo stop`

Stop the running container (immediate, no grace period).

### `silo rm`

Remove the container. Fails if the container is running unless `--force` is given.

| Flag | Description |
|---|---|
| `-f`, `--force` | Stop the container if it is running before removing |

### `silo rmi`

Remove the workspace image. With `--force`, also stops and removes the container first.

| Flag | Description |
|---|---|
| `-f`, `--force` | Stop and remove the container before removing the image |

### `silo user init`

Create user starter files under `$XDG_CONFIG_HOME/silo/` if they do not exist:

- `home-user.nix` — user home-manager config baked into the user image
- `silo.in.toml` — default values for new workspaces
- `devcontainer.in.json` — merged into every generated `.devcontainer.json`

### `silo user build`

Build the user image if it does not exist yet. The user image is shared across all workspaces.

### `silo user rmi`

Remove the user image.

### `silo status`

Print `Running` or `Stopped` for the workspace container.

### `silo devcontainer`

Generate a `.devcontainer.json` for VS Code in the current host directory. Does nothing if `.devcontainer.json` already exists. The generated container name is `<workspace-container-name>-dev`. See [VS Code devcontainer](#vs-code-devcontainer) for details.

### `silo devcontainer stop`

Stop the devcontainer.

### `silo devcontainer rm`

Remove the devcontainer. Fails if the container is running unless `--force` is given.

| Flag | Description |
|---|---|
| `--force`, `-f` | Stop the container if it is running before removing |

### `silo devcontainer status`

Print `Running` or `Stopped` for the devcontainer.

### `silo help`

Show the full command reference.

---

## Configuration

Configuration is TOML-based with two tiers. Later tiers override earlier ones:

1. Built-in defaults
2. User config at `$XDG_CONFIG_HOME/silo/silo.in.toml`
3. Workspace config at `.silo/silo.toml`
4. Runtime flags

On macOS, `~/.config/silo/` is used unless `$XDG_CONFIG_HOME` is set explicitly.

### Workspace config: `.silo/silo.toml`

Created automatically on first run. Seeded from `$XDG_CONFIG_HOME/silo/silo.in.toml` if present.

```toml
[general]
id             = "ab3f9c12"          # 8-char random ID; names container and image
user           = "alice"
container_name = "silo-ab3f9c12"
image_name     = "silo-ab3f9c12"

[connect]
command = "/bin/sh"                  # Command executed when connecting to container

[features]
shared_volume = false                # Mount shared volume at /silo/shared
nested        = false                # Allow nested Podman containers

[shared_volume]
paths = [
    "$HOME/.local/share/fish/fish_history",  # persist and share file
    "$HOME/.cache/uv/",                      # persist and share directory (trailing /)
]

[create]
extra_args = []                      # Extra arguments passed to podman create
```

**`[general]`**

| Key | Description |
|---|---|
| `id` | 8-character random alphanumeric workspace ID |
| `user` | Current username on the host |
| `container_name` | Name of the Podman container |
| `image_name` | Name of the workspace image |

**`[connect]`**

| Key | Default | Description |
|---|---|---|
| `command` | `/bin/sh` | Command executed when connecting to the container |

**`[features]`**

| Key | Default | Description |
|---|---|---|
| `shared_volume` | `false` | Mount the `silo-shared` Podman volume at `/silo/shared` inside the container |
| `nested` | `false` | Enable nested Podman (adds `--device /dev/fuse`, disables SELinux label) |

**`[shared_volume]`**

| Key | Description |
|---|---|
| `paths` | Paths inside the container backed by the shared volume. A trailing slash means directory; no trailing slash means file. `$HOME` is the only supported placeholder prefix. |

**`[create]`**

| Key | Default | Description |
|---|---|---|
| `extra_args` | `[]` | Extra arguments appended to `podman create`. Updated automatically when extra arguments are passed to `silo create`. |

### Workspace config: `.silo/home.nix`

Home-manager config applied only to this workspace's image. Created as an empty module on first run.

```nix
{ config, pkgs, ... }:
{
  home.packages = with pkgs; [
    nodejs
    python3
  ];
}
```

### User config: `$XDG_CONFIG_HOME/silo/`

| File | Description |
|---|---|
| `silo.in.toml` | Default values for new workspaces. `[general]` is ignored. |
| `home-user.nix` | User home-manager config baked into the user image. |
| `devcontainer.in.json` | Merged into every generated `.devcontainer.json`. |

See `examples/` for reference configs.

---

## How It Works

### Two-stage image build

silo builds two OCI images using Podman:

1. **User image** (`silo-<user>`) — shared across all workspaces. Alpine Linux with Nix and home-manager installed. The user `home-user.nix` is baked in here.
2. **Workspace image** (`silo-<id>`) — per-workspace, layered on top of the user image. The workspace `home.nix` is applied here.

Build context files are written to a temporary directory on the host and passed to `podman build`. No persistent build context is kept on disk.

### Workspace mount

The host directory is mounted into the container at `/workspace/<id>/<dirname>`, where `<id>` is the workspace ID and `<dirname>` is the host directory's basename.

### Shared volume

The named Podman volume `silo-shared` is mounted at `/silo/shared` inside every container. Data stored there — such as package caches — is shared across all workspaces and survives container restarts and image rebuilds.

For paths listed in `[shared_volume]`, a symlink inside the container pointing to `/silo/shared/` is created on every container start. A trailing slash marks a directory; no trailing slash marks a file. `$HOME` is expanded inside the container.

Example: `$HOME/.cache/uv/` creates a symlink from `$HOME/.cache/uv` to `/silo/shared/home/alice/.cache/uv` inside the container.
x
If a real file or directory already exists at the target path, the symlink is skipped and a warning is printed.

### Nix + home-manager

Each image build generates a Nix flake in a temporary directory on the host and passes it to `podman build`. The flake wires together `nixos-unstable`, home-manager, `home-user.nix` (user image), and `.silo/home.nix` (workspace image).

### VS Code devcontainer

`silo devcontainer` generates a `.devcontainer.json` on the host, pointing at the workspace image. The generated container name is `<workspace-container-name>-dev`. The user `$XDG_CONFIG_HOME/silo/devcontainer.in.json` is merged with the generated file:

- Objects merge recursively (key-by-key)
- Arrays concatenate (base array first, then input array)
- Scalars from input override base values

**Important**

- The `silo` container is independent from the devcontainer.
- Lifecycle is managed by VS Code/devcontainers, not by `silo`.
- `silo` commands (`start`/`stop`/`status`/`connect`/`rm`) target the regular workspace container.

Example `$XDG_CONFIG_HOME/silo/devcontainer.in.json`:

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
