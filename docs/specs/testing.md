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

## PTY-Backed Interactive Smoke Coverage

`make test-all` includes a narrow PTY-backed smoke path for `ww i`. These tests run in the host-based integration harness and are skipped by `make test` because short mode excludes all integration tests.

The PTY smoke harness starts the built `ww` binary with a pseudo-terminal attached to `stdin` and `stderr`, captures `stdout` separately, fixes `TERM=dumb` for deterministic accessible prompt rendering, writes key sequences into the terminal, and waits for stable prompt checkpoints with bounded deadlines. This coverage is intentionally small:

- starting `ww i` under a real TTY must pass the interactive TTY guard
- selecting `quit` from the real prompt loop must exit successfully without writing a path or other action result
- driving `list -> open` through the real prompt stack must exit successfully and emit exactly the selected absolute worktree path as the final stdout payload
- prompt rendering and terminal control output are treated as human UI and are not asserted with golden snapshots

The PTY tests prove prompt-library wiring, terminal input handling, and the interactive stdout/stderr contract at the integration boundary. Detailed branch logic remains covered by unit tests and non-PTY integration tests.
