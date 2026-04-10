# Interactive Mode Specification

## Overview

`ww i` is the Phase 4 human-oriented interactive entry point. It is a guided
prompt flow layered on top of the existing non-interactive CLI contract.

Interactive mode is an orchestration surface, not a separate capability
surface. Every externally observable action must map to an equivalent standard
`ww` command.

The implementation direction for this prompt flow is `huh`, matching the parent
plan and ADR choice for a lightweight grouped-step UI rather than a full-screen
TUI or ad hoc line prompts.

## Non-Negotiable Invariant

- Interactive mode must not expose a mutation or path-selection capability that
  cannot also be performed by a non-interactive `ww` command plus flags.
- If a future interactive flow needs behavior that lacks CLI parity, the
  non-interactive command must be specified and implemented first.

## Terminal and Stream Contract

- Prompt input uses `stdin`.
- Prompt rendering uses `stderr`.
- `stdout` is reserved for machine-consumable or path-only results produced by
  concrete interactive actions such as `open`.
- Successful `open` writes exactly one absolute path plus a trailing newline to
  `stdout` and does not add any surrounding label or decoration.
- Successful `remove`, `back`, and `quit` actions do not write to `stdout`.
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

The current implementation step defines:

- overview rendering
- top-level action selection
- shared action and session abstractions for later child flows
- the `list` child flow
- placeholder dispatch for `create` and `clean`
- `huh`-based prompt rendering for the implemented flows

The current implementation step does not implement the actual `create` or
`clean` interactive subflows yet.

## Placeholder Dispatch Contract

When the user selects one of the unimplemented flows:

- `create` prints:
  `Interactive create flow is not implemented yet. Use `ww create` for now.`
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

## List Flow Contract

Selecting `list` enters a worktree browser built from the same underlying
workspace/repo status computation used by `ww list`.

The list flow must:

- present worktrees, not repositories
- support case-insensitive filtering over:
  - repo name
  - branch name
  - status text including any `status_detail`
  - full absolute path
- display, for each visible candidate:
  - repo
  - branch
  - status
  - shortened path
  - main-worktree marker when applicable

The shortened path is a human-oriented display field only. Filtering still uses
the full absolute path.

The selector UX is keyboard-first:

- arrow keys and `j`/`k` navigate visible options
- `/` enters in-selector filtering over the visible option labels
- `q` exits the interactive session

After selecting a worktree, the action menu is:

- `open`
- `remove` when the selected worktree is not the main worktree
- `back`

Main worktrees remain selectable in the browser, but interactive mode must
clearly mark them and must not offer `remove`, matching the non-interactive
`ww remove` contract.

### `open`

- `open` is equivalent to `ww cd [--repo <repo>] <branch>`.
- On success, it writes only the selected path plus a trailing newline to
  `stdout`.
- All prompts, menus, context, and any human-readable guidance remain on
  `stderr`.
- After writing the path, the interactive session exits successfully.

### `remove`

- `remove` is equivalent to `ww remove [--repo <repo>] <branch>`.
- Before deletion, interactive mode shows a preview naming the selected
  worktree path and branch.
- `remove` requires explicit confirmation.
- On success, human-readable removal output is written to `stderr`, not
  `stdout`.
- After a successful removal, the session returns to the list browser so the
  user can continue browsing remaining worktrees.

## Future Child-Flow Contracts

The remaining child plans can implement:

- `create` as interactive orchestration over `ww create`
- `clean` preview/confirmation over `ww clean` / `ww clean --force`

Those child flows must preserve the same stream contract from this spec:
prompt UI on `stderr`, path-only or machine-oriented results on `stdout`.
