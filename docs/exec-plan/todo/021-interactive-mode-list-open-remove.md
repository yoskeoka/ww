# 021: Interactive Mode List, Open, and Remove

> **Execution**: Use `/execute-task` to implement this plan.

**Parent plan**: `docs/exec-plan/{todo,done}/interactive-mode-mvp.md`

**Depends on:** `020-interactive-mode-contract-and-foundation.md`

**Objective:** Implement the interactive worktree-selection flow for browsing existing worktrees, then support `open` and `remove` actions with strict parity to existing non-interactive commands.

## Required CLI Parity

| Interactive action | Equivalent CLI |
|--------------------|----------------|
| Select/filter worktrees | `ww list` as the data source |
| `open` selected worktree | `ww cd [--repo <repo>] <branch>` |
| `remove` selected worktree | `ww remove [--repo <repo>] <branch>` |

If implementation discovers a selected target that cannot be expressed as branch + optional repo, this plan must first add the missing non-interactive selector command/flag before completing the interactive action.

## Scope

### In Scope

- Build the workspace-wide worktree selector using existing list/status data
- Filter by repo, branch, status, and path
- Mark main worktrees clearly
- Support post-selection actions: `open`, `remove`, `back`
- Disable or omit `remove` for main worktrees
- Preserve `open` path-only stdout behavior

### Out of Scope

- Batch selection
- Multi-remove
- Top-level `remove` menu
- Custom fuzzy ranking beyond the chosen prompt library's filtering capabilities

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/cli-commands.md` | Expand `ww i` list-flow contract and parity references to `ww list`, `ww cd`, and `ww remove` |
| `docs/specs/interactive-mode.md` | Specify selector item fields, filtering dimensions, main-worktree handling, `open`/`remove` behavior, and stream-routing details |

## Code Changes

| File | Change |
|------|--------|
| `cmd/ww/sub_interactive.go` | Wire `list` action into the shared interactive session |
| `internal/interactive/` | Add list-flow controller, selector item formatting, action menu handling, and preview helpers |
| `integration_test.go` | Add interactive-mode tests for list/open/remove behavior where feasible |

## Design Notes

- Source data must come from the same underlying list/status logic used by `ww list`; do not fork status computation.
- `open` should not print any decoration around the path. The user must be able to use `ww i` open behavior in shell composition if they redirect streams appropriately.
- `remove` remains a confirmation-wrapped front-end over existing removal behavior.
- The main worktree is selectable for `open` but never removable, matching `ww remove`.

## Sub-tasks

- [ ] [parallel] Update specs for the list selector, open action, and remove action
- [ ] [depends on: specs] Build selector items from existing worktree list data
- [ ] [depends on: specs] Implement filterable worktree selection UI
- [ ] [depends on: selector] Implement post-selection actions: `open`, `remove`, `back`
- [ ] [depends on: actions] Implement main-worktree restrictions for `remove`
- [ ] [depends on: actions] Ensure `open` writes path-only `stdout` and keeps prompts/context on `stderr`
- [ ] [depends on: implementation] Add focused tests for selector formatting, filter input coverage, main-worktree restrictions, and `open` stream behavior

## Verification

- Interactive selection shows worktrees, not repos
- Selector items display repo, branch, status, shortened path, and main-worktree marker
- Filtering works across repo, branch, status, and full path
- `open` returns exactly the selected path on `stdout`
- `remove` is unavailable for the main worktree
- The resulting `open` and `remove` actions are each achievable via existing non-interactive commands
