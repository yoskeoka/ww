# 012: Fix Docker Exec Output Handling in Integration Tests

## Objective

Make `make test-all` pass reliably now that Docker-backed integration tests are exercised in normal development and CI.

## Problem

`internal/testutil.ContainerEnv.Exec` assumes container exec output is always Docker-multiplexed and asks `testcontainers-go` to demux it. On the current Docker/testcontainers combination, the reader can already be plain combined output, causing every integration test to fail with `Unrecognized input header`.

## Spec Changes

- Add a testing spec that documents:
  - `make test` runs short/unit tests only
  - `make test-all` runs the full suite including Docker integration tests
  - Docker-backed integration helpers must capture combined command output without depending on a single exec stream encoding

## Code Changes

- Update `internal/testutil/container.go` to read container exec output in a way that works whether the reader is already plain combined output or still Docker-multiplexed.
- Keep existing integration test behavior and assertions unchanged.

## Verification

- `go test -run TestVersionCommand -v .`
- `make test-all`
