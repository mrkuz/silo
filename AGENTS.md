# AGENTS.md

## Build and Test

```bash
go build .              # Build binary
go test ./...           # Run all tests
go test ./internal      # Run internal package tests
go test ./features      # Run feature spec tests
go test -run TestName   # Run a single test
go vet ./...            # Run static analysis
```

## Testing Constraints

**Do not use `t.Parallel()`** — tests call `os.Chdir` to a temp workspace, which is process-global. Use `SetupWorkspace` for chdir and cleanup.

The `execCommand` seam in `internal/container.go` is mocked via `MockExecCommand(t, responses)`. Responses are keyed by the full command string.

## Template Path Resolution

Templates in `internal/templates/` are embedded via go:embed for both development and installed binaries.

## Config Field Naming

Config structs use `Persistence` with `SharedPaths` (TOML tag `toml:"persistence"` / `shared_paths`). The old names (`SharedVolume`, `Paths`) no longer exist.

## Persistence Volume Conventions

- Volume name: `silo` (not `silo-shared`)
- Mount point: `/silo/persistence`
- Shared paths at: `/silo/persistence/shared/<container-path>`
- Subpath in mount string: `shared/<container-path>`

## TOML Style

Match `examples/silo.user.toml`:
- Keys have no leading spaces
- Array elements use exactly 2-space indent
- Blank line between tables

## Lifecycle

```
init → build → create → start → connect
```

`start` internally calls `EnsureCreated` and `VolumeSetup`. See `cmd/start.go:Start` and `internal/config.go:EnsureStarted`.

## Only External Dependency

`github.com/BurntSushi/toml` — config parsing uses strict mode (unknown keys cause errors).