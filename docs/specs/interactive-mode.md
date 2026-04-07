# Interactive Mode Specification

## Overview

`ww i` is the Phase 4 human-oriented interactive entry point. It is a guided
prompt flow layered on top of the existing non-interactive CLI contract.

Interactive mode is an orchestration surface, not a separate capability
surface. Every externally observable action must map to an equivalent standard
`ww` command.

## Non-Negotiable Invariant

- Interactive mode must not expose a mutation or path-selection capability that
  cannot also be performed by a non-interactive `ww` command plus flags.
- If a future interactive flow needs behavior that lacks CLI parity, the
  non-interactive command must be specified and implemented first.

## Terminal and Stream Contract

- Prompt input uses `stdin`.
- Prompt rendering uses `stderr`.
- `stdout` is reserved for machine-consumable or path-only results produced by
  concrete interactive actions such as future `open`.
- In the foundation step, no successful interactive action writes to `stdout`
  because `create`, `list`, and `clean` are still placeholders.
- `ww i --json` is rejected immediately; interactive mode does not provide JSON
  output.

## Entry Preconditions

- `stdin` must be a TTY.
- `stderr` must be a TTY.
- If either precondition fails, `ww i` exits non-zero with:
  `interactive mode requires a TTY on stdin and stderr; use standard ww commands and see ww --help`

This keeps human-facing prompt output off `stdout`, preserving the shell
contract introduced in Phase 3.

## Context Model

Interactive mode reuses the existing workspace detection and repo resolution
rules. No alternate workspace model is allowed.

The overview screen displays:

- detected mode: `single-repo` or `workspace`
- root path: repo root in single-repo mode, workspace root in workspace mode
- repo names when in workspace mode

## Top-Level Actions

The MVP top-level action menu is fixed:

- `create`
- `list`
- `clean`
- `quit`

The foundation step implements:

- overview rendering
- top-level action selection
- shared action and session abstractions for later child flows
- placeholder dispatch for `create`, `list`, and `clean`

The foundation step does not implement the actual `create`, `list`, or `clean`
interactive subflows yet.

## Placeholder Dispatch Contract

When the user selects one of the unimplemented flows:

- `create` prints:
  `Interactive create flow is not implemented yet. Use `ww create` for now.`
- `list` prints:
  `Interactive list flow is not implemented yet. Use `ww list` for now.`
- `clean` prints:
  `Interactive clean flow is not implemented yet. Use `ww clean` for now.`
- After printing the placeholder guidance, the session returns to the
  top-level menu instead of exiting immediately.

Selecting `quit` exits successfully without writing to `stdout`.

## Shared Action Model

The shared foundation defines stable action identifiers for the top-level menu:

- `create`
- `list`
- `clean`
- `quit`

Later child plans must reuse these identifiers rather than inventing separate
string forms for prompts, tests, and dispatch.

## Future Child-Flow Contracts

The foundation exists so later child plans can implement:

- `create` as interactive orchestration over `ww create`
- `list` browsing over `ww list`, with `open` parity to `ww cd` and `remove`
  parity to `ww remove`
- `clean` preview/confirmation over `ww clean` / `ww clean --force`

Those child flows must preserve the stream contract from this spec: prompt UI
on `stderr`, path-only or machine-oriented results on `stdout`.
