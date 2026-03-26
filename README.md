# silo

A personal developer sandbox built on Alpine Linux, Nix, and home-manager. Running `silo.sh` from any directory drops you into a reproducible, fully-configured  environment inside a rootless Podman container, with your working directory mounted automatically.

## How it works

Each directory gets its own named container. The first time you run `silo.sh` from a directory, a `.silo.toml` file is created with a random ID that links the working directory to its container (`silo-<id>`) and workspace mount (`/workspace/<id>/<dirname>`).

On subsequent runs, `silo.sh` joins a running container via `podman exec`, restarts a stopped one, or creates a new one — transparently.

A shared named volume (`silo-state`) is mounted at `/state/global` across all containers. Selected `$HOME` paths are symlinked into it, so state like shell history and tool caches is shared between containers and survives restarts.

## Requirements

- [Podman](https://podman.io/)

## Installation

Clone the repository and link `silo.sh` onto your `PATH`:

```bash
git clone https://github.com/mrkuz/silo.git
ln -s silo/silo.sh /usr/local/bin/silo.sh
```

## Usage

### Customization

Before first use, edit `home.nix` to configure packages and settings, then build the image:

```bash
./build.sh
```

Key variables (username, git config, ...) are defined in the `vars` block in `flake.nix`. Rebuild with `./build.sh` after any changes to `home.nix` or `Dockerfile`.

To share additional paths across containers via the `silo-state` volume, add `persist` calls to `entrypoint.sh`. Use a trailing `/` for directories:

```bash
persist ".config/myapp/"      # directory
persist ".config/myapp.conf"  # file
```

### Running

**Start or join a container** for the current directory:

```bash
silo.sh
```

**Flags:**

| Flag | Effect |
|------|--------|
| `-p` | Enable nested Podman (relaxes security opts, adds `/dev/fuse`) |
| `-W` | Disable workspace mount (current directory will not be mounted) |
| `-S` | Disable state volume mount |

Any arguments after `--` are passed directly to `podman run`:

```bash
silo.sh -- --env MY_VAR=value --publish 8080:8080
```

**Remove the container** for the current directory:

```bash
silo.sh rm
```

### Development Container (VS Code)

To generate a `.devcontainer.json` for the current directory, run:

```bash
silo.sh devcontainer
```

This writes a `.devcontainer.json` that points at the `silo` image, sets the remote user, and configures the integrated terminal.

The generated `runArgs` mirror the security options that would apply in the current context (default hardened, or relaxed when `-p` is passed).

> **Note:** The devcontainer and the silo container are independent. Running `silo.sh` always creates or starts its own named container - it does not join or reuse a container started by VS Code. The `silo-state` volume is not mounted in devcontainers.