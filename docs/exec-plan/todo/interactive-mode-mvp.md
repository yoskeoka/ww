# Interactive Mode MVP (`ww i`)

> **Execution**: Use `/execute-task` to implement this plan.

**Objective:** Add a human-oriented interactive mode to `ww` via `ww i`, using a lightweight prompt flow that preserves workspace-wide visibility while guiding common operations for direct terminal users.

**Depends on:** Project-plan update for Phase 4 / FR-24 (PR #71) merging to `main`

## Context

Phase 2 established `ww`'s main strength: workspace-wide visibility across repositories. The interactive mode must preserve that advantage instead of forcing users into repo-first navigation.

The desired MVP is:

- command name: `ww i`
- lightweight prompt flow, not a full-screen TUI
- initial screen shows detected mode and repo context, then asks which operation to run
- menu scope: `create`, `list`, `clean`, `quit`
- `list` operates on **worktrees**, not repos, with interactive filtering
- `list` actions: `open`, `remove`, `back`
- `open` writes the selected path to `stdout` only
- `remove` is available only from `list`; there is no top-level `remove` menu
- `clean` shows a repo-level summary first, then a detailed confirmation view
- batch or multi-select operations are explicitly out of scope

## Reviewed Decisions / Constraints

Past decisions reviewed before planning:

- `docs/design-decisions/core-beliefs.md`: AI-first remains intact because `ww i` is additive; non-interactive commands remain the primary agent/scripting interface.
- `docs/design-decisions/adr.md`: existing git-native and workspace-aware decisions still apply. The interactive mode must call the same underlying logic as standard subcommands rather than inventing separate behavior.

New design choices captured by this plan:

- use `ww i` rather than `ww ui` / `ww interactive`
- use a lightweight prompt library rather than Bubble Tea for the MVP
- prefer `huh` over `promptui` / `survey`
  - `survey` is out because its official README states it is no longer maintained
  - `promptui` is viable for one-off prompts, but this feature is a multi-step guided flow
  - `huh` fits grouped step flows better and already includes filterable selection behavior suitable for the first finder implementation
- do not implement a custom fuzzy-ranking engine in Phase 4 MVP; use filterable selection and revisit only if the experience is insufficient

## Scope

### In Scope

- New `ww i` subcommand
- TTY-only execution
- Mode/repo overview before operation selection
- Guided `create` flow
- Guided `list` flow with filterable worktree selection
- Guided `clean` flow with preview/confirmation
- Path-only `stdout` result for `open`
- Explicit handling of main worktrees in list results

### Out of Scope

- Auto-entering interactive mode when `ww` is run without subcommands
- Full-screen TUI / Bubble Tea app
- Batch or multi-select actions
- A top-level `remove` menu
- Custom fuzzy-scoring or fzf parity
- JSON mode support inside `ww i`

## UX Specification

### Entry / Preconditions

- `ww i` requires an interactive terminal.
- In non-TTY environments, fail immediately with a clear error that tells the user to use standard commands and consult `ww --help`.
- Workspace detection uses the existing Phase 2 logic. No separate workspace model is introduced.

### Initial Screen

Before asking for the action, show lightweight context:

- detected mode: `workspace` or `single-repo`
- current workspace root or repo root
- repo list when in workspace mode

Then prompt for one action:

- `create`
- `list`
- `clean`
- `quit`

### Create Flow

1. Resolve target repo:
   - workspace mode: choose repo interactively
   - single-repo mode: use the current repo without prompting
2. Enter or select branch information
3. Show preview:
   - target worktree path
   - base branch
   - copy/symlink actions
   - hook execution when configured
4. Confirm
5. Execute using the existing create logic

### List Flow

1. Build the workspace-wide worktree list using existing list/status logic
2. Present a filterable selector over **worktrees**
3. Display fields:
   - repo
   - branch
   - status
   - shortened path
   - main-worktree marker when applicable
4. Search/filter against:
   - repo
   - branch
   - status
   - full path
5. After selecting a worktree, offer:
   - `open`
   - `remove`
   - `back`
6. Main worktree behavior:
   - `open` allowed
   - `remove` disabled / not offered
   - UI clearly marks it as the main worktree
7. `open` behavior:
   - print the selected path to `stdout` only
   - keep human-readable guidance off `stdout`
8. `remove` behavior:
   - show preview before deletion
   - require confirmation
   - reuse the existing remove logic

### Clean Flow

1. Compute cleanable worktrees using existing merged/stale logic
2. Show a repo-level summary first, including zero-count repos when in workspace mode
3. Ask whether to proceed, and whether removal is safe or forced
4. Show a detailed list of targeted worktrees before final confirmation
5. Execute using the existing clean logic

The clean flow is intentionally more informative than plain `ww clean`; the interactive mode's value is visibility and confirmation.

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/cli-commands.md` | Add `ww i` command behavior, TTY requirement, and high-level flows |
| `docs/specs/interactive-mode.md` | New spec describing interactive-mode UX, list selector behavior, clean summary/detail flow, and `open` output contract |

## Design Decision Changes

| File | Change |
|------|--------|
| `docs/design-decisions/adr.md` | Record the Phase 4 MVP interaction model: `ww i`, `huh`, filterable worktree selector, no batch/multi-select |

## Code Changes

| File | Change |
|------|--------|
| `cmd/ww/main.go` | Register the new `i` subcommand |
| `cmd/ww/sub_interactive.go` | New interactive-mode entry point and high-level flow wiring |
| `cmd/ww/helpers.go` | Reuse or extend shared repo/workspace resolution helpers as needed |
| `internal/interactive/` | New package for prompt flow, selection formatting, and TTY checks |
| `integration_test.go` | Add interactive-mode integration coverage where feasible |

Exact file layout may shift during implementation, but the interactive flow should live outside `worktree/` so the core business logic remains reusable.

## Testing Strategy

Interactive flows should not depend solely on manual testing.

- Extract prompt/session logic behind small interfaces so the step transitions are unit-testable without a real terminal.
- Keep git/worktree behavior delegated to existing tested code paths.
- Add at least one non-TTY integration test to verify `ww i` fails with the intended message.
- Add focused tests for:
  - workspace initial screen inputs
  - worktree selector item formatting / filtering data
  - main worktree action restrictions
  - `open` returning path-only `stdout`
  - clean summary and detailed confirmation construction

If full PTY integration proves too expensive in the first pass, prioritize unit-level flow coverage plus non-TTY command verification.

## Sub-tasks

- [ ] [parallel] Add specs for `ww i` in `docs/specs/cli-commands.md` and new `docs/specs/interactive-mode.md`
- [ ] [parallel] Append ADR entry documenting `ww i`, `huh`, and the MVP scope boundaries
- [ ] [depends on: specs, ADR] Add CLI wiring for `ww i` and non-TTY rejection
- [ ] [depends on: specs] Implement the initial overview screen and top-level action selection
- [ ] [depends on: initial overview] Implement the guided `create` flow
- [ ] [depends on: initial overview] Implement the `list` flow with filterable worktree selection and `open` / `remove` actions
- [ ] [depends on: initial overview] Implement the `clean` flow with repo summary and detailed confirmation
- [ ] [depends on: create, list, clean] Add unit and integration coverage for the interactive flow helpers and command behavior

## Verification

- `ww i` starts only in interactive terminals
- Workspace mode preserves repo-wide visibility before action selection
- `list` filters worktrees, not repos
- Main worktrees are selectable for `open` and unavailable for `remove`
- `open` writes only the selected path to `stdout`
- `clean` shows summary + detailed confirmation before execution
- Existing non-interactive commands remain unchanged
