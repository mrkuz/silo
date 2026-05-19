# silo

Per-workspace development containers. Powered by Podman, Nix, and home-manager.

## Goals

- Simple way to create and run personal workspace containers, set up via home-manager

## Non-Goals

- Full secured agent sandbox
- Reproducibility — builds may vary across machines or time

---

## Features

- **Per-workspace isolation** — each workspace gets its own container
- **Nix + home-manager** — shared `home.user.nix` and per-workspace `.silo/home.nix`
- **Workspace mount** — the host directory is mounted inside the container
- **Persistence** — persist package caches and other data across containers and rebuilds
- **Port forwarding** — expose container ports to the host
- **Resource limits** — control CPU, memory, and process limits per workspace
- **Nested Podman** — optional support for running containers inside the container
- **VS Code support** — generate a `.devcontainer.json` which uses the workspace image

---

## Build and Install

**Requirements:** Go 1.23+, Podman

```bash
# Build binary
go build .

# Run tests
go test ./...

# Install to $GOPATH/bin
go install .
```

---

## Quick Start

```bash
silo
```

On first run, `silo` initializes required files, builds the workspace image, starts the container, and connects to it. Subsequent runs skip steps that are already complete and connect directly.

### Customize workspace container

```bash
# Edit .silo/home.nix
silo stop
silo build --rebuild
silo
```

---

## Lifecycle

Every workspace goes through a fixed chain of steps. Each step depends on the ones before it. Running `silo` triggers the full chain.

```
user init → init → build → volume setup → start → connect
```

| Step | Description | Idempotency | Calls |
|---|---|---|---|
| **user init** | Creates user config files under `$XDG_CONFIG_HOME/silo/` | Writes files only on first run | - |
| **init** | Creates `.silo/silo.toml` and `.silo/home.nix` | Writes files only on first run | `silo user init` |
| **build** | Builds the workspace image | Skipped if image exists | `silo init` |
| **volume setup** | Creates directories on shared volume | Safe to re-run | - |
| **start** | Creates and starts the container | Skipped if container is already running | `silo build`,  `silo volume setup` |
| **connect** | Opens an interactive shell inside the running container | - | - |

---

## Commands

```
silo [--stop]
silo init [--podman|--no-podman]
silo build [--rebuild] [--no-cache]
silo start
silo volume setup
silo connect
silo stop
silo rm
silo status
silo user init
silo devcontainer [--update]
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

Initialize workspace files. Creates `.silo/silo.toml` and `.silo/home.nix`.

| Flag | Description |
|---|---|
| `--podman` | Enable Podman inside the container |
| `--no-podman` | Disable Podman inside the container |

### `silo build`

Build the workspace image if it does not exist yet.

| Flag | Description |
|---|---|
| `--rebuild` | Force rebuild even if image exists; aborts if container exists or is running |
| `--no-cache` | Disable build cache |

### `silo start`

Creates and starts the container. If the container is already running, this command does nothing.

### `silo volume setup`

Creates directories on the persistence volume for paths configured in `[persistence]`. Runs a temporary container with the workspace image for the setup.

### `silo connect`

Connect to the container for the current workspace. Requires the container to exist and be running.

### `silo stop`

Stop and remove the running container.

### `silo rm`

Remove the workspace image. If the container exists and is stopped, it is removed first. Returns an error if the container is running.

### `silo user init`

Create user files under `$XDG_CONFIG_HOME/silo/` if they do not exist:

- `home.user.nix` - User-wide home-manager config baked into every workspace image
- `silo.user.toml` - User-wide silo configuration
- `devcontainer.user.json` - Personal configuration merged into every generated `.devcontainer.json`

### `silo status`

Print `Running` or `Stopped` for the workspace container.

### `silo devcontainer`

Generates `.silo/devcontainer.json` and `.devcontainer.json` for VS Code in the current host directory. See [VS Code devcontainer](#vs-code-devcontainer) for details.

| Flag | Description |
|---|---|
| `--update` | Update existing `.devcontainer.json` |

### `silo devcontainer stop`

Stop and remove the devcontainer.

### `silo devcontainer status`

Print `Running` or `Stopped` for the devcontainer.

### `silo devcontainer connect`

Connect to the devcontainer for the current workspace. Requires the devcontainer to exist and be running.

### `silo help`

Show the full command reference.

---

## Configuration

silo is configured via two TOML files. Both configs are merged at runtime.

1. User config at `$XDG_CONFIG_HOME/silo/silo.user.toml`
2. Workspace config at `.silo/silo.toml`

If `$XDG_CONFIG_HOME` is not set, `~/.config/silo/` is used.

Configuration changes require restart of the container to become effective.

### Sections

| Section | `silo.user.toml` | `.silo/silo.toml` | Description |
|---|---|---|---|
| **`[general]`** | `user`<br>- | -<br>`id` | Your username<br>Workspace ID (8-char random) |
| **`[features]`** | - | `podman` | Enable nested Podman inside the container |
| **`[persistence]`** | `shared_paths`<br>`private_paths` | `shared_paths`<br>`private_paths` | Paths persisted and shared across all containers<br>Private paths to persist<br>A trailing slash marks a directory |
| **`[podman]`** | `create_args` | `create_args` | Podman create arguments<br>Default values are set on `silo init` |
| **`[network]`** | `ports` | `ports` | Port forwarding mappings (e.g. `8080:8080`) |
| **`[limits]`** | - | `cpus`<br>`memory`<br>`processes` | CPU limit<br>Memory limit (MB)<br>Process ID limit<br>Zero or negative = unlimited |

### Merge behavior

- **`[general].user`** - user config only
- **`[general].id`** - workspace config only; set once on first run
- **`[features].podman`** - workspace config only; set by `silo init --[no-]podman` on first run
- **`[persistence].shared_paths`** - user paths first, then workspace paths
- **`[persistence].private_paths`** - user paths first, then workspace paths
- **`[podman].create_args`** - user args first, then workspace args
- **`[network].ports`** - user ports first, then workspace ports

### Example: `$XDG_CONFIG_HOME/silo/silo.user.toml`

```toml
[general]
user = "alice"

[persistence]
shared_paths = [
    "$HOME/.cache/uv/"
]
private_paths = [
    "$HOME/.local/share/fish/fish_history"
]

[podman]
create_args = []

[network]
ports = []
```

### Example: `.silo/silo.toml`

```toml
[general]
id = "ab3f9c12"

[features]
podman = false

[persistence]
shared_paths = []
private_paths = []

[podman]
create_args = [
  "--cap-drop=ALL",
  "--cap-add=NET_BIND_SERVICE",
  "--security-opt",
  "no-new-privileges"
]

[network]
ports = [ "8080:8080" ]

[limits]
cpus = 0
memory = 0
processes = 1024
```

## How It Works

### Image build

silo builds a workspace image using Podman. The image is based on Fedora, with Nix and home-manager installed.

Each image build generates a Nix flake in a temporary directory on the host. The flake wires together `nixos-unstable`, home-manager, `home.user.nix`, and `.silo/home.nix`.

### Container creation

When creating a container, silo passes several arguments to `podman create`:

- **Name and hostname**: `--name silo-<id>` and `--hostname silo-<id>`
- **Workspace mount**: The host directory is mounted at `/workspace/<id>/<dirname>`
- **Persistence volume**: The named volume `silo` is mounted at `/silo/persistence`
- **Limits args**: `--cpus=N`, `--memory=Nm`, `--pids-limit=N` (from `[limits]` config)
- **Network ports**: Port forwarding mappings from `[network].ports`
- **Create args**: Additional args from `[podman].create_args`

### Persistence volume

silo uses named Podman volumes with subpath mounts. Each configured path gets its own mount point directly inside the container, using the volume's subpath feature.

**Shared paths**: Persisted at `shared/<path>` on the volume. Shared across all workspaces. A trailing slash marks a directory; no trailing slash marks a file. `$HOME` is expanded inside the container.

_Example:_ `shared_paths = ["$HOME/.cache/uv/"]` creates a volume mount with `target=/home/alice/.cache/uv` and `subpath=shared/home/alice/.cache/uv`.

**Private paths**: Persisted at `<SILO_ID>/<path>` on the volume. Local to each workspace.

_Example:_ `private_paths = ["/data"]` in workspace `abc12345` creates a volume mount with `target=/data` and `subpath=abc12345/data`.

### Nested Podman

When the Podman feature is enabled, Podman is installed and configured inside the container. This is implemented as home-manager module, which is part of the image and enabled via `silo.podman.enable = true` in `.silo/home.nix`

### VS Code devcontainer

`silo devcontainer` generates `.silo/devcontainer.json` and `.devcontainer.json`. The latter is created by merging three sources in the following order:

1. **Personal defaults** - `$XDG_CONFIG_HOME/silo/devcontainer.user.json`
2. **Project-specific** - `.silo/devcontainer.json`
3. **Silo** - Sets workspace image, username, container name, mounts, limits, forwarded ports and run args

Merge behavior:
- Objects merge recursively (key-by-key)
- Arrays concatenate
- Scalars override

**Important:**

- The devcontainer uses the workspace image, but is independent from the silo container
- The container is named `silo-<id>-dev`
- Creation is handled by VS Code/devcontainers, not by `silo`.
