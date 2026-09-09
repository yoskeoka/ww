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

The named fixture profile, benchmark command, repetition count, and separate per-command limits live in `tools/performance-budget.json`. Budget values use human-managed milliseconds per command operation (`max_ms_per_op`); an operation is one completed `ww list` or `ww clean --dry-run` process invocation. Limits are calibrated on the GitHub-hosted runner and are therefore CI budgets, not portable local wall-clock promises. Its default profile can be scaled for investigation without changing code by setting `WW_PERF_REPOS` and `WW_PERF_WORKTREES_PER_REPO`; the latter must remain at least two so the status shape stays representative. `WW_PERF_CLEANABLE_WORKTREES_PER_REPO` optionally increases the number of merged targets per repository for clean-heavy, same-host investigation; it defaults to one and must leave at least one upstream-tracking active worktree. This override does not change the default profile, benchmark names, repetition count, or CI budgets. Developers may collect profiles with `make perf-profile PERF_PROFILE_DIR=/tmp/ww-perf-profile` and inspect them with a compatible pprof viewer.

Pull requests run the performance budget as a dedicated advisory job. A budget exceedance makes that job red and prints the observed value, configured budget, and reproduction command; it prompts investigation of the candidate change or a cumulative regression, but is not a required branch-protection check. The job summary includes the benchmark comparison and identifies whether the configured budget was met, exceeded, or could not be evaluated. The normal `test` job remains independent. If the initial PR measurement is consistently slower than one minute, a separate issue must consider daily measurement and automatic issue creation rather than changing this contract silently.

### Patch-equivalence investigation profile

Set `WW_PERF_PATCH_HEAVY=1` with the normal scale overrides to create a supplementary, non-budgeted fixture. In this profile, secondary worktree branches each contain two non-empty commits whose combined changes are squash-integrated into the base after the branches are created. This deliberately bypasses the commit-level `git cherry` fast path and exercises aggregate patch comparison across branches sharing a merge-base. It is for same-host before/after evidence only: it does not alter the default fixture, benchmark names, repetition count, or CI budgets.

### Persistent discovery-cache profiles

All unit, integration, and performance tests redirect `os.UserCacheDir` inputs
to test-owned temporary state. Tests must not populate or depend on the
operator's real cache.

The existing budgeted `BenchmarkWorkspaceList` and
`BenchmarkWorkspaceCleanDryRun` profiles remain cold-cache regression gates:
their cache state is reset before timing and their fixture, command boundary,
names, repetitions, and versioned limits do not change. Non-budgeted warm
variants use the same PR #280 fixture and command boundary, prime the cache
outside the timed region, and verify the same list entries or cleanable count.
Warm measurements are same-host evidence for material improvement, not a new
CI budget or a portable latency promise.

### Positive remote-cache coverage

Remote-cache tests use a test-owned user-cache directory and a deterministic
clock. Unit coverage must include a complete positive hit, one requested branch
missing from the set, the exact 30-second TTL boundary, expiry, changed remote
name/URL/common-directory identity, corrupt or tampered entries, credential
non-persistence, and concurrent atomic writers. Git tests cover multiple
configured remote URL values and live-query failures without exposing URL
contents in cache data or diagnostics.

Worktree status tests must prove that a cached positive keeps a present branch
`active`, that remote deletion is detected as `stale` after a miss or expiry,
that cache failures preserve the live error path, and that no cached state can
produce a false `merged`, `stale`, or `cleanable` result. Separate-process
integration tests cover hit, candidate miss, expiry, deletion delay followed by
live stale detection, URL changes, remote errors, and unavailable/read-only
cache fallback.

The performance fixture keeps the existing cold budget profiles unchanged. It
also provides non-budgeted warm profiles that prime positive remote evidence
outside the timer, plus an opt-in deterministic delayed-remote profile via
`WW_PERF_REMOTE_DELAY_MS`. Warm and delayed profiles must record remote probe
counts and elapsed timings at the same PR #280 fixture scale; they are
same-host evidence and do not replace the cold regression gate.

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
