# Remove Docker dependency from integration tests

**Type:** improvement | **Priority:** Medium

## Problem

The integration test suite runs inside a shared Docker container (`testcontainers-go`), but the isolation it provides is largely redundant. Each test already gets a unique temp directory via `MkdirTemp`, and `ww`'s parent-directory traversal stops at `/tmp/` where no `.ww.toml` or `.git` exists. The Docker layer adds complexity and is itself the source of flakiness (see `docs/issues/done/docker-integration-parallel-flakiness.md`).

### What Docker provides vs. alternatives

| Role | Docker approach | Host-native alternative |
|------|----------------|------------------------|
| Filesystem isolation | Container boundary | `MkdirTemp` with random suffix (already used) |
| Git config isolation | `GIT_CONFIG_GLOBAL=/tmp/gitconfig` in container | Set `GIT_CONFIG_GLOBAL` to a test-scoped temp file |
| Prevent host side-effects | All ops in container | Tests only operate in temp dirs under `/tmp/` |
| Consistent environment | `golang:1.23` image | CI runner spec (e.g., GitHub Actions `runs-on`) |

### Costs of Docker

- **Flakiness source**: Docker exec contention under parallel test execution caused the flakiness that required serializing 13 tests
- **Environment friction**: Rancher Desktop requires `DOCKER_HOST` and `TESTCONTAINERS_RYUK_DISABLED` configuration
- **Startup overhead**: Container pull, binary cross-compile for Linux, binary copy into container
- **Code complexity**: `readCombinedOutput` with Docker stream multiplexing detection, `ContainerEnv` abstraction layer
- **Dependency**: Docker daemon must be running for `make test-all`

## Proposed Approach

Replace `ContainerEnv` (Docker-based) with a `HostEnv` that:

1. Builds `ww` for the host OS/arch (no cross-compile needed)
2. Creates temp directories via `os.MkdirTemp`
3. Runs `ww` and `git` as host processes (`os/exec`)
4. Sets `GIT_CONFIG_GLOBAL` to a test-scoped temp file to prevent host git config leakage
5. Adds deep path nesting if needed for workspace detection safety: `/tmp/ww-test-XXXXXX/sandbox/repo/`

### Migration path

1. Introduce `HostEnv` implementing the same interface as `ContainerEnv`
2. Switch `TestMain` to use `HostEnv` by default
3. Verify all tests pass on host (multiple runs)
4. Remove `ContainerEnv`, `testcontainers-go` dependency, and Docker stream handling code
5. Optionally: re-enable `t.Parallel()` on previously serialized tests (exec contention no longer applies with host processes)

## Success Criteria

- `make test-all` works without Docker running
- No environment-specific configuration needed (`DOCKER_HOST`, `TESTCONTAINERS_RYUK_DISABLED`)
- Test suite is at least as fast as current Docker-based suite
- All previously parallel tests can run parallel again
- No host git config contamination during test runs
