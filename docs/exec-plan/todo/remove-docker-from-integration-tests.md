# Remove Docker from Integration Tests
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Close the remaining documentation and issue-tracking gap for the Docker-to-host integration test migration.

The code migration itself appears to have already landed in `47827f6` / PR #81 (`refactor: replace Docker integration tests with host-native execution`): the current `main` has `internal/testutil.HostEnv`, `docs/specs/testing.md` describes host-based integration tests, and `go.mod` no longer references `testcontainers-go`. However, `docs/issues/remove-docker-from-integration-tests.md` still exists as an active issue while an almost identical resolved copy also exists at `docs/issues/done/remove-docker-from-integration-tests.md`. The ADR log also still records the earlier Docker/testcontainers decision without a later superseding decision.

This plan resolves that inconsistency so the repo memory matches the actual test architecture.

## Context Read

- `docs/project-plan.md`: `ww` prioritizes a fast, portable Go CLI with minimal runtime friction. Host-native tests align with portability and speed.
- `docs/design-decisions/core-beliefs.md`: spec-code parity and correctness come before speed; stable working code should not be refactored for aesthetics. This plan should avoid unnecessary harness rewrites.
- `docs/design-decisions/adr.md`: the 2026-03-19 ADR chose `testcontainers-go` for integration test isolation. Current code and specs have since moved away from that choice, so the ADR needs an explicit superseding entry rather than silent contradiction.
- `docs/specs/testing.md`: already specifies host-based integration tests and says no Docker daemon or container runtime is required.

## Current Evidence

- `internal/testutil/host.go` exists and builds/runs `ww` as a host process.
- `integration_test.go` uses `*testutil.HostEnv`.
- `rg "testcontainers|ContainerEnv|readCombinedOutput|stdcopy"` finds no live code references.
- `go.mod` / `go.sum` no longer include `testcontainers-go`.
- Active and done copies of `remove-docker-from-integration-tests.md` both exist under `docs/issues/`.

## Code Changes

No intentional production or test harness code changes are expected.

During execution, verify that no residual Docker/testcontainers implementation remains in:

- `go.mod`
- `go.sum`
- `integration_test.go`
- `internal/testutil/`
- `.github/workflows/` test jobs that run `make test` / `make test-all`

If verification reveals leftover Docker-specific test harness code, remove it only if it is directly part of the old integration-test harness. Do not modify unrelated workflow Docker usage such as GitHub Actions sandbox/firewall infrastructure.

## Spec Changes

No behavioral spec rewrite is expected because `docs/specs/testing.md` already describes the desired host-based harness.

During execution:

- Confirm `docs/specs/testing.md` still states that `make test-all` runs without Docker.
- Confirm `docs/spec-code-mapping.md` still maps `docs/specs/testing.md` to `internal/testutil/` and `integration_test.go`.
- Update these files only if verification finds drift.

## Documentation and Issue Changes

- Remove the stale active issue at `docs/issues/remove-docker-from-integration-tests.md` after confirming the resolved copy in `docs/issues/done/remove-docker-from-integration-tests.md` is present and accurate.
- If the active copy has better wording than the done copy, update the done copy before deleting the active duplicate.
- Append a new ADR entry superseding the 2026-03-19 Docker/testcontainers decision:
  - context: Docker/testcontainers was originally chosen for isolation, but it introduced flakiness, environment friction, startup overhead, and exec-stream complexity.
  - decision: integration tests run host-native through `HostEnv`, with per-test temp dirs and test-scoped `GIT_CONFIG_GLOBAL`.
  - consequences: no Docker daemon needed for `make test-all`, less harness complexity, host/CI environment parity must be managed through Go and Git process isolation instead of container boundaries.

## Sub-tasks

- [ ] [parallel] Verify live code and dependency state has no old Docker/testcontainers harness leftovers.
- [ ] [parallel] Verify `docs/specs/testing.md` and `docs/spec-code-mapping.md` still match the current host-based harness.
- [ ] [parallel] Reconcile the duplicate active/done issue files, keeping only the resolved issue under `docs/issues/done/`.
- [ ] [depends on: verification] Append a superseding ADR entry for host-native integration tests.
- [ ] [depends on: issue reconciliation, ADR update] Run formatting/lint-free documentation checks applicable to the repo.
- [ ] [depends on: verification] Run `make test-all` to prove the host-native integration suite still passes without Docker-specific setup.

## Parallelism

The code/dependency scan, spec scan, and issue-file reconciliation can happen independently. The ADR entry should wait until verification confirms the live state, so it records facts rather than assumptions.

## Design Decisions

This execution should not make a new test architecture decision; it should document the architecture already implemented by PR #81.

Past decision: the 2026-03-19 ADR chose `testcontainers-go` because it preserved `go test` as the single test framework while isolating integration tests from host config and filesystem state. The later implementation achieved the same single-framework goal with less infrastructure by using `HostEnv`, temp directories, and scoped Git config. The ADR update should explicitly supersede the earlier decision and explain why the project changed course.

## Verification

- `rg -n "testcontainers|ContainerEnv|readCombinedOutput|stdcopy" go.mod go.sum integration_test.go internal/testutil docs/specs docs/design-decisions`
- `make test`
- `make test-all`

`make test-all` is the key verification gate because the success criteria for the original issue was that the full integration suite works without Docker.

## Success Criteria

- No active duplicate remains at `docs/issues/remove-docker-from-integration-tests.md`.
- `docs/issues/done/remove-docker-from-integration-tests.md` remains as the resolved issue record.
- `docs/design-decisions/adr.md` contains a dated entry superseding the old Docker/testcontainers integration-test decision.
- Testing specs and spec-code mapping still match the host-native harness.
- `make test-all` passes without Docker-specific environment setup.
