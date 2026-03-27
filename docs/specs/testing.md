# Testing Specification

## Targets

| Command | Behavior |
|---|---|
| `make test` | Runs `go test -short ./...` and skips integration tests |
| `make test-all` | Runs `go test ./...` and includes integration tests |

## Host-Based Integration Harness

Integration tests execute `ww` and supporting shell commands directly on the host machine. Each test gets its own temporary directory via `os.MkdirTemp` for filesystem isolation.

The test harness (`HostEnv`) provides:

- **Binary management**: Builds `ww` once for the host OS/arch and reuses the binary across all tests.
- **Git config isolation**: Sets `GIT_CONFIG_GLOBAL` to a test-scoped temporary file, preventing host git config contamination during test runs.
- **Filesystem helpers**: `MkdirTemp`, `MkdirAll`, `WriteFile`, `ReadFile`, `PathExists`, `IsSymlink` operate directly on the host filesystem.
- **Command execution**: `Exec`, `Git`, `RunWW` run commands as host processes via `os/exec`, combining stdout and stderr.

All tests may run in parallel. No Docker daemon or container runtime is required.
