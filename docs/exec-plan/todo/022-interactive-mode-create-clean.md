# 022: Interactive Mode Create and Clean

> **Execution**: Use `/execute-task` to implement this plan.

**Parent plan**: `docs/exec-plan/{todo,done}/interactive-mode-mvp.md`

**Depends on:** `020-interactive-mode-contract-and-foundation.md`

**Objective:** Implement guided `create` and `clean` flows that improve human visibility and confirmation while remaining thin wrappers over non-interactive `ww create` and `ww clean`.

## Required CLI Parity

| Interactive action | Equivalent CLI |
|--------------------|----------------|
| Confirmed create | `ww create [--repo <repo>] <branch>` |
| Confirmed clean (safe) | `ww clean` |
| Confirmed clean (force) | `ww clean --force` |

If the interactive flow starts requiring parameters or behaviors not expressible with the commands above, this plan must first extend the non-interactive CLI and spec before exposing that behavior in `ww i`.

## Scope

### In Scope

- Guided repo selection for `create` in workspace mode
- Branch entry for `create`
- Pre-execution preview for `create`
- Repo-summary-first `clean` UX
- Detailed clean target confirmation
- Choice between safe and forced clean execution

### Out of Scope

- Interactive branch creation policy beyond what `ww create` already does
- Batch/multi-repo create in one confirmation step
- Adding extra clean filters or target narrowing not already expressible non-interactively

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/cli-commands.md` | Expand `ww i` create/clean flow contract and parity references to `ww create` and `ww clean` |
| `docs/specs/interactive-mode.md` | Specify create preview contents, clean summary/detail flow, and confirmation semantics |

## Code Changes

| File | Change |
|------|--------|
| `cmd/ww/sub_interactive.go` | Wire `create` and `clean` actions into the shared interactive session |
| `internal/interactive/` | Extend the shared `huh`-based prompt session with create-flow controller, clean-flow controller, preview builders, and confirmation helpers |
| `integration_test.go` | Add interactive-mode tests for create/clean behavior where feasible |

## Design Notes

- `create` preview should show only data already derivable from the existing create path: target path, base branch, copy/symlink actions, and configured hooks.
- `clean` summary is a UX layer over the existing cleanability computation. Do not introduce a second cleanability algorithm.
- Force-vs-safe clean choice must map directly onto `ww clean` vs `ww clean --force`.
- Workspace mode should preserve repo-wide visibility by showing zero-count repos in the clean summary.
- Use the same `huh`-based prompt surface chosen by the parent plan and ADR, rather than introducing a second interactive UI style for `create`/`clean`.
- Reuse the shared `huh` keymap and grouped-step flow style so arrows / `j` / `k` navigation, `q` to quit, and confirmation behavior remain consistent across interactive mode.

## Sub-tasks

- [ ] [parallel] Update specs for create and clean interactive flows
- [ ] [depends on: specs] Implement repo selection and branch entry for create
- [ ] [depends on: create input] Implement create preview and confirmation
- [ ] [depends on: create preview] Execute create via existing command/business logic path
- [ ] [depends on: specs] Implement cleanable worktree summary grouped by repo
- [ ] [depends on: clean summary] Implement safe/force choice and detailed confirmation view
- [ ] [depends on: clean confirmation] Execute clean via existing command/business logic path
- [ ] [depends on: implementation] Add focused tests for create preview data, clean summary generation, force/safe selection, and parity-sensitive execution paths

## Verification

- In workspace mode, `create` prompts for repo selection before branch entry
- In single-repo mode, `create` skips repo selection
- `create` preview matches the non-interactive create inputs and derived effects
- Clean summary includes zero-count repos in workspace mode
- Clean confirmation shows the detailed target list before execution
- Safe and forced clean choices map to the same results as `ww clean` and `ww clean --force`
