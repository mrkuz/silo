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
- **Nix + home-manager** — shared `home.user.nix` and per-workspace `.silo/home.nix`
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
init → build → create → volume setup → start → connect
```

| Step | Description | Output | Idempotency |
|---|---|---|---|
| **init** | Creates `.silo/silo.toml`, `.silo/home.nix`; runs `silo user init` to create user files | Workspace + user files | Writes config only on first run |
| **build** | Ensures user image exists, then builds workspace image if needed | Container image | Images are cached; only missing ones are built |
| **create** | Creates the container if it doesn't exist | Container (stopped) | Skipped if container already exists |
| **volume setup** | Creates directories on the shared volume | Configured container | Safe to re-run |
| **start** | Starts the container if not running | Running container | Skipped if container is already running |
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
silo [--stop]
silo init [--podman|--no-podman]
silo build [-f|--force]
silo start
silo volume setup
silo connect
silo stop
silo rm
silo status
silo user init
silo user build [-f|--force]
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

### `silo init`

Initialize workspace files. Creates `.silo/silo.toml` and `.silo/home.nix`, then delegates to `silo user init` for user files. Writes config only on first run. If a flag is not provided, the default from `silo.user.toml` is used; if that's also unset, built-in defaults apply.

| Flag | Description |
|---|---|
| `--podman` | Enable Podman inside the container |
| `--no-podman` | Disable Podman inside the container |

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

Remove the workspace image. If the container exists and is stopped, it is removed first. Returns an error if the container is running; in that case neither the container nor the image is modified.

### `silo user init`

Create user starter files under `$XDG_CONFIG_HOME/silo/` if they do not exist:

- `home.user.nix` — user home-manager config baked into the user image
- `silo.user.toml` — default values for new workspaces
- `devcontainer.user.json` — merged into every generated `.devcontainer.json`

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

### `silo devcontainer status`

Print `Running` or `Stopped` for the devcontainer.

### `silo devcontainer connect`

Connect to the devcontainer for the current workspace. Requires the devcontainer to exist and be running.

### `silo help`

Show the full command reference.

---

## Configuration

Configuration is TOML-based with three tiers. Later tiers override earlier ones:

1. Built-in defaults
2. User config at `$XDG_CONFIG_HOME/silo/silo.user.toml`
3. Workspace config at `.silo/silo.toml`

On macOS, `~/.config/silo/` is used unless `$XDG_CONFIG_HOME` is set explicitly.

The two config files serve different purposes:

| | `silo.user.toml` | `.silo/silo.toml` |
|---|---|---|
| **Purpose** | Defaults for new workspaces; shared across all workspaces | Per-workspace runtime config |
| **`[general]`** | `user` — your username | `id` — workspace ID (8-char random) |
| **`[features]`** | — | `podman` — enable nested Podman |
| **`[shared_volume]`** | `paths` — default paths | `paths` — additional paths (merged) |
| **`[podman]`** | `create_args` — prepended | `create_args` — base args |

### Merge behavior

- **`[general].user`** — from user config only
- **`[general].id`** — from workspace config only; set once on first run
- **`[features].podman`** — from workspace config only; set by `silo init --[no-]podman`
- **`[shared_volume].paths`** — merged: user paths first, then workspace paths
- **`[podman].create_args`** — merged: user args prepended to workspace args

### User config: `$XDG_CONFIG_HOME/silo/silo.user.toml`

Default values for new workspaces. Your username and default shared volume paths live here. `[general].id` is ignored.

```toml
[general]
user = "alice"

[shared_volume]
paths = [
    "$HOME/.cache/uv/",                      # persist and share directory (trailing /)
    "$HOME/.local/share/fish/fish_history",  # persist and share file
]

[podman]
create_args = []
```

### Workspace config: `.silo/silo.toml`

Per-workspace runtime config. Created automatically on first run.

```toml
[general]
id = "ab3f9c12"

[features]
podman = false

[shared_volume]
paths = [
    "$HOME/.local/share/fish/fish_history",  # persist and share file
    "$HOME/.cache/uv/",                      # persist and share directory (trailing /)
]

[podman]
create_args = [
  "--cap-drop=ALL",
  "--cap-add=NET_BIND_SERVICE",
  "--security-opt",
  "no-new-privileges"
]
```

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

### User config files: `$XDG_CONFIG_HOME/silo/`

| File | Description |
|---|---|
| `silo.user.toml` | Default values for new workspaces |
| `home.user.nix` | User home-manager config baked into the user image |
| `devcontainer.user.json` | Merged into every generated `.devcontainer.json` |

See `examples/` for reference configs.

---

## How It Works

### Two-stage image build

silo builds two OCI images using Podman:

1. **User image** (`silo-<user>`) — shared across all workspaces. Fedora with Nix and home-manager installed. The user `home.user.nix` is baked in here.
2. **Workspace image** (`silo-<id>`) — per-workspace, layered on top of the user image. The workspace `home.nix` is applied here.

Build context files are written to a temporary directory on the host and passed to `podman build`. No persistent build context is kept on disk.

### Workspace mount

The host directory is mounted into the container at `/workspace/<id>/<dirname>`, where `<id>` is the workspace ID and `<dirname>` is the host directory's basename.

### Shared volume

The named Podman volume (`silo-shared`) is mounted at `/silo/shared` inside every container. Data stored there — such as package caches — is shared across all workspaces and survives container restarts and image rebuilds.

For paths listed in `[shared_volume]`, a subpath mount is created inside the container. A trailing slash marks a directory; no trailing slash marks a file. `$HOME` is expanded inside the container.

Example: `$HOME/.cache/uv/` creates a volume mount with `target=/home/alice/.cache/uv` and `subpath=home/alice/.cache/uv`.

### Nested Podman

When `--podman` is passed to `silo init`, Podman is installed and configured inside the container, allowing you to run containers within the container. This is useful for testing containerized workflows or running Docker-in-Docker style setups.

The `silo.podman.enable = true` option is set in `.silo/home.nix` when `--podman` is used, which activates the Podman service via home-manager.

### Nix + home-manager

Each image build generates a Nix flake in a temporary directory on the host and passes it to `podman build`. The flake wires together `nixos-unstable`, home-manager, `home.user.nix` (user image), and `.silo/home.nix` (workspace image).

### VS Code devcontainer

`silo devcontainer` generates a `.devcontainer.json` on the host, pointing at the workspace image. The generated container name is `<workspace-container-name>-dev`. The user `$XDG_CONFIG_HOME/silo/devcontainer.user.json` is merged with the generated file:

- Objects merge recursively (key-by-key)
- Arrays concatenate (base array first, then input array)
- Scalars from input override base values

**Important**

- The `silo` container is independent from the devcontainer.
- Lifecycle is managed by VS Code/devcontainers, not by `silo`.
- `silo` commands (`start`/`stop`/`status`/`connect`/`rm`) target the regular workspace container.

Example `$XDG_CONFIG_HOME/silo/devcontainer.user.json`:

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
