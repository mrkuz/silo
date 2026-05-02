# silo

Per-directory developer containers, powered by Podman, Nix, and home-manager.

## Goals

- Simple way to create and run personal workspace containers, configured via home-manager

## Non-Goals

- Full secured agent sandbox
- Deterministic reproducibility — builds may vary across machines or time
- Sharability — workspaces are personal and local to a user

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

# Run tests
go test ./...

# Install to $GOPATH/bin
go install .
```

---

## Lifecycle

Every workspace goes through a fixed chain of steps. Each step depends on the ones before it. Running `silo` (or `silo connect`) triggers the full chain automatically.

```
init → build → setup → start → connect
```

| Step | Description | Output | Idempotency |
|---|---|---|---|
| **init** | Creates `.silo/silo.toml`, `.silo/home.nix`; runs `silo user init` to create user files | Workspace + user files | Writes config only on first run |
| **build** | Ensures user image exists, then builds workspace image if needed | Container image | Images are cached; only missing ones are built |
| **start** | Creates the container if needed, then starts it | Running container | Skipped if container is already running |
| **setup** | Creates directories on the shared volume using a temporary container | Configured container | Safe to re-run |
| **connect** | Opens an interactive shell inside the running container | Terminal session | — |

You can run individual steps:

```bash
silo init          # Initialize workspace and user files
silo build         # Build images (if not already built)
silo start         # Start container (creates if needed)
silo volume setup  # Run shared volume setup
silo connect       # Connect to container (triggers missing steps automatically)
```

---

## Commands

```
silo [--stop] [-- args...]
silo init [--podman|--no-podman] [--shared-volume|--no-shared-volume]
silo build
silo start
silo volume setup
silo connect
silo stop
silo rm [-f|--force]
silo status
silo user init
silo user build
silo user rm
silo devcontainer
silo devcontainer connect
silo devcontainer stop
silo devcontainer status
silo help
```

### `silo` (default)

Run the full lifecycle chain if needed, then connect to the container for the current workspace.

| Flag | Description |
|---|---|
| `--stop` | Stop and remove the container when the session exits |
| `-- ...` | Pass remaining arguments to `podman exec` |

### `silo init`

Initialize workspace files. Creates `.silo/silo.toml` and `.silo/home.nix`, then delegates to `silo user init` for user files. Writes config only on first run. If a flag is not provided, the default from `silo.in.toml` is used; if that's also unset, built-in defaults apply.

| Flag | Description |
|---|---|
| `--podman` | Enable Podman inside the container |
| `--no-podman` | Disable Podman inside the container |
| `--shared-volume` | Enable shared volume |
| `--no-shared-volume` | Disable shared volume |
| `-f`, `--force` | Overwrite existing workspace files |

### `silo build`

Ensure the user image exists, then build the workspace image if it does not exist yet.

| Flag | Description |
|---|---|
| `-f`, `--force` | Force rebuild workspace image; aborts if container exists or is running |

### `silo start`

Start the container and run post-start setup. Creates the container if it doesn't exist. If the container is already running, this command does nothing.

### `silo volume setup`

Creates directories on the shared volume for paths configured in `[shared_volume]`. Runs a temporary container with the user image — the workspace container does not need to be running. This step runs automatically after every start.

### `silo connect`

Connect to the container for the current workspace. Requires the container to exist and be running. Does not trigger build, create, or start steps.

### `silo stop`

Stop and remove the running container (immediate, no grace period).

### `silo rm`

Remove the workspace image. With `--force`, also stops and removes the container first.

| Flag | Description |
|---|---|
| `-f`, `--force` | Stop and remove the container before removing the image |

### `silo user init`

Create user starter files under `$XDG_CONFIG_HOME/silo/` if they do not exist:

- `home-user.nix` — user home-manager config baked into the user image
- `silo.in.toml` — default values for new workspaces
- `devcontainer.in.json` — merged into every generated `.devcontainer.json`

| Flag | Description |
|---|---|
| `-f`, `--force` | Overwrite existing user files |

### `silo user build`

Build the user image if it does not exist yet. The user image is shared across all workspaces.

| Flag | Description |
|---|---|
| `-f`, `--force` | Force rebuild user image |

### `silo user rm`

Remove the user image.

### `silo status`

Print `Running` or `Stopped` for the workspace container.

### `silo devcontainer`

Generate a `.devcontainer.json` for VS Code in the current host directory. Does nothing if `.devcontainer.json` already exists. The generated container name is `<workspace-container-name>-dev`. See [VS Code devcontainer](#vs-code-devcontainer) for details.

| Flag | Description |
|---|---|
| `-f`, `--force` | Overwrite existing `.devcontainer.json` |

### `silo devcontainer stop`

Stop and remove the devcontainer (immediate, no grace period).

| Flag | Description |
|---|---|
| `--force`, `-f` | Stop the container if it is running before removing |

### `silo devcontainer status`

Print `Running` or `Stopped` for the devcontainer.

### `silo devcontainer connect`

Connect to the devcontainer for the current workspace. Requires the devcontainer to exist and be running.

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
podman        = false                # Enable Podman inside the container

[shared_volume]
name  = "silo-shared"                        # Podman volume name (default: silo-shared)
paths = [
    "$HOME/.local/share/fish/fish_history",  # persist and share file
    "$HOME/.cache/uv/",                      # persist and share directory (trailing /)
]

[create]
arguments = [
  "--cap-drop=ALL",
  "--cap-add=NET_BIND_SERVICE",
  "--security-opt",
  "no-new-privileges"
]
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
| `shared_volume` | `false` | Enable the shared volume |
| `podman` | `false` | Enable Podman inside the container |

**`[shared_volume]`**

| Key | Default | Description |
|---|---|---|
| `name` | `silo-shared` | Shared volume name. |
| `paths` | `[]` | Paths inside the container backed by the shared volume. A trailing slash means directory; no trailing slash means file. `$HOME` is the only supported placeholder prefix. |

**`[create]`**

| Key | Default | Description |
|---|---|---|
| `arguments` | computed | Arguments appended to `podman create`. Set by `silo init` based on enabled features. User-provided arguments in `silo.in.toml` are prepended. |

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

The named Podman volume (default: `silo-shared`, configurable via `[shared_volume].name`) is mounted at `/silo/shared` inside every container. Data stored there — such as package caches — is shared across all workspaces and survives container restarts and image rebuilds.

For paths listed in `[shared_volume]`, a subpath mount is created inside the container. A trailing slash marks a directory; no trailing slash marks a file. `$HOME` is expanded inside the container.

Example: `$HOME/.cache/uv/` creates a volume mount with `target=/home/alice/.cache/uv` and `subpath=home/alice/.cache/uv`.

### Nested Podman

When `--podman` is passed to `silo init`, Podman is installed and configured inside the container, allowing you to run containers within the container. This is useful for testing containerized workflows or running Docker-in-Docker style setups.

The `module.podman.enable = true` option is set in `.silo/home.nix` when `--podman` is used, which activates the Podman service via home-manager.

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
