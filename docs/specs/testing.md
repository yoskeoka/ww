# Testing Specification

## Targets

| Command | Behavior |
|---|---|
| `make test` | Runs `go test -short ./...` and skips integration tests |
| `make test-all` | Runs `go test ./...` and includes integration tests |
| `make perf-check` | Repeats the command-boundary `ww list` and `ww clean --dry-run` benchmarks and compares their median elapsed time with the versioned advisory budget |
| `make perf-profile PERF_PROFILE_DIR=<path>` | Writes standard Go CPU and memory profiles for the performance fixture to the requested directory |

## Workspace Command Performance Coverage

Performance coverage runs `ww list` and `ww clean --dry-run` against a host-native, non-sandbox workspace fixture. The fixture uses real child Git repositories, linked secondary worktrees, and local bare remotes; it includes merged and upstream-tracking active branches. Fixture construction is outside the benchmark timer, so each `ns/op` value covers CLI startup through completed command output only.

The named fixture profile, benchmark command, repetition count, and separate per-command limits live in `tools/performance-budget.json`. Its default profile can be scaled for investigation without changing code by setting `WW_PERF_REPOS` and `WW_PERF_WORKTREES_PER_REPO`; the latter must remain at least two so the status shape stays representative. Developers may collect profiles with `make perf-profile PERF_PROFILE_DIR=/tmp/ww-perf-profile` and inspect them with `go tool pprof /tmp/ww-perf-profile/cpu.pprof` or `go tool pprof /tmp/ww-perf-profile/mem.pprof`.

Pull requests run the performance budget as a dedicated advisory job. A budget exceedance makes that job red and prints the observed value, configured budget, and reproduction command; it prompts investigation of the candidate change or a cumulative regression, but is not a required branch-protection check. The normal `test` job remains independent. If the initial PR measurement is consistently slower than one minute, a separate issue must consider daily measurement and automatic issue creation rather than changing this contract silently.

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
