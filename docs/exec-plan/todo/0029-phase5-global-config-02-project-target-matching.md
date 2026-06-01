# 0029: Phase 5 Global Config Project Target Matching

> **Execution**: Use `/execute-task` to implement this plan.

**Parent plan series**: `0028-phase5-global-config-01-foundation.md` -> `0029-phase5-global-config-02-project-target-matching.md` -> `0030-phase5-global-config-03-materialization-profiles.md`

## Objective

Add deterministic project-target matching on top of the Phase 5 global-config foundation so one user-owned config can apply different settings to different repositories or workspace members without relying on committed repo-local `.ww.toml` files.

## Context

- `docs/project-plan.md` defines FR-28 as global project-target matching using the main worktree as the stable target anchor.
- The current config model is repository-scoped only. Once global config exists, `ww` needs a predictable way to select project-specific settings before the worktree manager runs.
- Matching should stay simple enough for humans and agents to audit. Hidden fallback behavior or ambiguous best-match logic would conflict with the current repo-local simplicity.

## Options and Trade-offs

1. **Ordered project blocks with first-match-wins against the main worktree root (recommended)**
   - Pros: Deterministic and easy to read. Supports intentional override ordering without inventing specificity scoring rules.
   - Cons: Large configs may require careful ordering by the user.
2. **Automatic most-specific-match-wins**
   - Pros: Can reduce ordering burden in some cases.
   - Cons: Specificity rules become another policy surface. Harder to explain when two patterns partially overlap.
3. **Single exact-path match only**
   - Pros: Very simple.
   - Cons: Too rigid for the portability goal. Fails the use case of sharing settings across multiple similar repositories.

**Recommendation**: Option 1. Match global project blocks in declared order against the main worktree root and stop at the first match. Keep the initial matcher narrow and explicit; add more sophisticated selection only if real usage demands it.

## Scope

### In Scope

- Define the project-target matching syntax and evaluation order
- Anchor matching on the repository main worktree root rather than the current secondary worktree path
- Document failure/ignore behavior for invalid or unmatched target blocks
- Add tests for overlapping rules and deterministic selection

### Out of Scope

- Materialization profile syntax itself; that belongs to Phase 5-03
- Implicit combination of multiple matching project blocks
- Recursive workspace policy engines or organization-wide config inheritance

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/configuration.md` | Add the global project-target block format, match anchor, ordering rule, and unmatched behavior |
| `docs/specs/git-operations.md` | Clarify that target matching anchors on the repository main worktree root used for git operations |
| `docs/project-plan.md` | Mark FR-28 status only if this child plan completes the full matching contract |

## Design Decision Changes

| File | Change |
|------|--------|
| `docs/design-decisions/adr.md` | Append ADR for project-target matching anchor and selection rule if the foundation ADR does not already cover it |

## Code Changes

| File | Change |
|------|--------|
| `internal/config/config.go` | Extend the global config schema and selection logic for project-target blocks |
| `internal/config/config_test.go` | Add target-matching tests, including overlapping and unmatched cases |
| `cmd/ww/main.go` | Ensure the matched project layer resolves before the manager config is finalized |

## Test Changes

- Add tests proving:
  - matching uses the main worktree root as the stable anchor
  - first matching project block wins
  - unmatched blocks leave the base global config unchanged
  - invalid project-target definitions fail clearly instead of silently half-applying

## Design Notes

- The target anchor must be stable across `ww create`, `ww cd`, and invocation from secondary worktrees; the repository main root is the least surprising anchor.
- Avoid implicit multi-match merging in this phase. One selected project block is easier to explain and debug.
- Keep the initial matcher expressive enough for path-based reuse, but do not invent a broad policy DSL.

## Sub-tasks

- [ ] [parallel] Specify the project-target syntax and first-match-wins rule in `docs/specs/configuration.md`
- [ ] [parallel] Decide whether `docs/specs/git-operations.md` needs an explicit anchor clarification
- [ ] [depends on: 0028 foundation] Implement project-target parsing and selection in `internal/config`
- [ ] [depends on: implementation] Thread selected project settings into final manager config resolution
- [ ] [depends on: implementation] Add regression tests for overlapping rules, unmatched targets, and secondary-worktree invocation

## Verification

- `go test ./internal/config ./cmd/ww`
- Manual verification:
  - one global config applies different settings to two repositories in the same workspace
  - invoking `ww` from a secondary worktree still matches the intended project using the main repo root
  - overlapping project rules select the first declared match consistently
