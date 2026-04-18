# 023: Interactive Mode PTY Smoke Tests

> **Execution**: Use `/execute-task` to implement this plan.

**Parent context**: `020-interactive-mode-contract-and-foundation.md`, `021-interactive-mode-list-open-remove.md`, and `022-interactive-mode-create-clean.md` in `docs/exec-plan/{todo,done}/`

**Objective:** Add a minimal PTY-backed integration test path for `ww i` so the project verifies at least one real interactive terminal session in CI-like test runs, without replacing the existing unit-heavy interactive test strategy.

## Problem

The current interactive-mode coverage proves the command contract and flow logic, but it stops short of exercising a real TTY-backed prompt session:

- integration tests verify help, `--json` rejection, and non-TTY failure behavior
- unit tests cover interactive flow transitions behind fake session/UI interfaces
- no test currently launches `ww i` behind a PTY, sends keys, and verifies that the real `huh`-driven prompt loop behaves correctly

That leaves a gap around prompt-library wiring, terminal input handling, and stream behavior that can regress without mechanical detection.

## Scope

### In Scope

- Add a PTY-capable integration-test helper for launching `ww i`
- Add one smoke test that starts interactive mode and quits cleanly
- Add one smoke test that drives `list -> open` and verifies path-only `stdout`
- Keep the coverage intentionally narrow and deterministic
- Preserve the existing unit-test-first strategy for detailed interactive logic

### Out of Scope

- Full-screen snapshot testing of interactive rendering
- Exhaustive PTY coverage for `create`, `clean`, and `remove`
- Golden tests for terminal escape sequences or cursor motion
- Replacing existing fake-UI/unit coverage with PTY-heavy tests
- Introducing a second interactive library or non-Go test runner

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/testing.md` | Document PTY-backed interactive smoke coverage, when it runs, and what behavior it is expected to prove |
| `docs/specs/interactive-mode.md` | Tighten the verification story for `quit` and `list -> open` under a real TTY-backed session, including the expectation that prompt UI stays on `stderr` while `open` writes only the selected path to `stdout` |

## Code Changes

| File | Change |
|------|--------|
| `internal/testutil/` | Add a PTY-oriented helper for spawning `ww` with controllable stdin/stdout/stderr and reading terminal output with bounded waits |
| `integration_test.go` or a new focused interactive integration test file | Add PTY smoke tests for `quit` and `list -> open` |
| `go.mod` / `go.sum` | Add the PTY dependency if needed (`creack/pty` or equivalent) |

## Design Notes

- Follow the existing ADR direction: interactive mode remains a lightweight prompt layer over normal CLI behavior, and detailed branch logic stays covered by unit tests.
- Prefer a small, explicit PTY harness in Go over shell `expect` scripts or a second-language test runner so the integration suite stays in one toolchain.
- Keep assertions high-signal and resilient. Verify stable user-visible checkpoints rather than every line of terminal paint.
- The first PTY tests should target the most basic end-to-end confidence checks:
  - `quit`: the prompt loop starts under a real TTY and exits successfully
  - `list -> open`: real prompt navigation reaches the list flow, selects a worktree, triggers `open`, and emits only the selected path on `stdout`
- If `huh` rendering proves timing-sensitive, centralize the waiting/retry logic in the helper rather than scattering sleeps through test bodies.
- Do not weaken the current non-TTY integration tests; PTY smoke coverage is additive.

## Past Decisions

Relevant prior decisions from `docs/design-decisions/adr.md`:

- Interactive mode is a lightweight prompt flow with strict non-interactive CLI parity.
- Human-readable prompt context belongs on `stderr`, and path-oriented output belongs on `stdout`.
- The foundation should remain unit-testable without a real terminal.

Apply the same reasoning here: add only enough PTY integration to catch real-terminal wiring regressions while keeping most behavioral coverage in the existing unit-level flow tests.

## Sub-tasks

- [x] [parallel] Update `docs/specs/testing.md` with PTY smoke-test expectations and limits
- [x] [parallel] Update `docs/specs/interactive-mode.md` with PTY-backed verification expectations for `quit` and `list -> open`
- [x] [depends on: specs] Choose and wire a Go PTY helper approach for integration tests
- [x] [depends on: PTY helper] Add a `ww i` smoke test that starts the real interactive session and exits via `quit`
- [x] [depends on: PTY helper] Add a `ww i` smoke test that drives `list -> open` and verifies path-only `stdout`
- [x] [depends on: PTY helper] Make the PTY harness robust against bounded rendering delays without relying on arbitrary long sleeps
- [x] [depends on: implementation] Run the relevant integration tests and confirm they pass reliably in the host-based test harness

## Verification

- A PTY-backed test can start `ww i` without tripping the non-TTY guard
- The `quit` smoke test exits successfully after real key-driven interaction
- The `list -> open` smoke test reaches the list flow through the real prompt stack
- `open` still emits exactly the selected path on `stdout`, with prompt UI remaining off `stdout`
- Existing non-TTY interactive tests continue to pass
- Existing unit tests for interactive flow logic remain the primary detailed behavior coverage
