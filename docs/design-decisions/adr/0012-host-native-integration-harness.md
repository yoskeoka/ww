# Integration tests: host-native harness supersedes testcontainers-go

## Status

Accepted on 2026-04-20. Supersedes [0007](0007-testcontainers-integration-harness.md).

## Context

The 2026-03-19 ADR chose testcontainers-go so integration tests could stay in Go's standard test framework while isolating filesystem and git configuration state from the host. That approach worked, but the Docker layer introduced flakiness under parallel execution, local environment friction, startup overhead, and exec-stream complexity that were disproportionate for a single-binary CLI.

The current test harness now provides the same core isolation properties with host processes:

- each test operates under a per-test temporary directory
- git configuration is scoped with a test-owned `GIT_CONFIG_GLOBAL`
- `ww` and `git` execute through Go's normal `os/exec` process model

## Decision

Integration tests run host-native through `internal/testutil.HostEnv`. `make test-all` runs unit and integration tests without requiring a Docker daemon, container runtime, `DOCKER_HOST`, or testcontainers-specific setup.

This supersedes the 2026-03-19 testcontainers-go decision for the integration test harness. Docker use that belongs to unrelated GitHub Actions sandbox or firewall infrastructure is outside this decision.

## Consequences

- **Positive**: `make test-all` has less environmental friction for local development and CI.
- **Positive**: The harness no longer needs Docker stream multiplexing or container exec plumbing.
- **Positive**: Integration tests still use Go's standard test runner and can run in parallel with per-test filesystem and git config isolation.
- **Negative**: Host and CI parity now depends on Go process isolation, temp directories, and scoped git config rather than container boundaries.
