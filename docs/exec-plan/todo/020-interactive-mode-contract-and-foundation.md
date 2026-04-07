# 020: Interactive Mode Contract and Foundation

> **Execution**: Use `/execute-task` to implement this plan.

**Parent plan**: `docs/exec-plan/{todo,done}/interactive-mode-mvp.md`

**Objective:** Freeze the Phase 4 interactive-mode MVP contract before feature work begins, then add the shared `ww i` command foundation that later create/list/clean flows can reuse.

## Non-Negotiable Rule

Interactive mode must not introduce unique executable behavior.

- Any action in `ww i` that causes an external effect MUST map to an equivalent non-interactive `ww` command plus flags.
- If implementation reveals missing parity, extend the non-interactive CLI first in this plan or a dependent child plan before exposing the interactive action.

## Scope

### In Scope

- Add/extend specs for `ww i`
- Append ADR for interaction model and parity rule
- Register `ww i`
- Enforce TTY preconditions
- Establish stdout/stderr routing contract for interactive mode
- Implement the initial overview screen and top-level action selection
- Introduce shared interfaces/helpers so subsequent flows are testable without a real terminal

### Out of Scope

- Implementing the actual create/list/clean subflows beyond stub wiring
- Adding new interactive-only capabilities
- Full PTY end-to-end coverage if unit-level seam coverage is sufficient for this foundation step

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/cli-commands.md` | Add `ww i` command definition, TTY rules, failure behavior, top-level actions, and the interactive execution parity rule |
| `docs/specs/interactive-mode.md` | New spec for prompt model, stream routing, overview screen, shared action model, and child-flow contracts |

## Design Decision Changes

| File | Change |
|------|--------|
| `docs/design-decisions/adr.md` | Append ADR: `ww i` is a lightweight prompt flow, uses `huh`, and cannot expose capabilities lacking non-interactive CLI parity |

## Code Changes

| File | Change |
|------|--------|
| `cmd/ww/main.go` | Register `i` subcommand |
| `cmd/ww/sub_interactive.go` | Add command entry point, TTY validation, overview screen, and top-level dispatch |
| `cmd/ww/helpers.go` | Reuse or extend repo/workspace resolution helpers if needed |
| `internal/interactive/` | New package for session abstraction, terminal capability checks, rendering helpers, and top-level action model |
| `integration_test.go` | Add non-TTY failure coverage and foundation-level command behavior tests where appropriate |

## Design Notes

- Reuse existing workspace detection and repo resolution. No alternate workspace model is allowed.
- Treat interactive mode as an orchestration layer over existing command/business logic.
- Preserve the Phase 3 shell contract: path-only success output to `stdout`, human-readable context to `stderr`.
- `ww i --json` MUST fail fast with a clear error directing users to standard non-interactive commands.
- The top-level action menu is fixed at `create`, `list`, `clean`, `quit` for the MVP.

## Sub-tasks

- [ ] [parallel] Update `docs/specs/cli-commands.md` with `ww i` contract and parity rule
- [ ] [parallel] Add `docs/specs/interactive-mode.md`
- [ ] [parallel] Append ADR entry for the Phase 4 interaction model and parity rule
- [ ] [depends on: specs, ADR] Add `cmd/ww/sub_interactive.go` and register `ww i` in `cmd/ww/main.go`
- [ ] [depends on: specs] Implement TTY validation and `--json` rejection
- [ ] [depends on: specs] Implement the initial overview screen and top-level action selection stub/dispatch
- [ ] [depends on: implementation] Add unit tests for terminal checks, stream routing decisions, and action dispatch
- [ ] [depends on: implementation] Add at least one integration test proving non-TTY invocation fails with the intended message

## Verification

- `ww i` is registered and visible in help
- `ww i --json` fails fast with a clear non-interactive guidance message
- `ww i` fails when no interactive terminal is available
- With redirected `stdout`, interactive prompts still use `stderr` and `stdout` remains reserved for path-only action results
- The top-level menu shows `create`, `list`, `clean`, `quit`
- Specs and ADR explicitly state the parity rule
