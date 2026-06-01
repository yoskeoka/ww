# 0029: Phase 5 project match rules

> **Execution**
> Run `/execute-task` for this plan.

**Parent plan series**
`0028-phase5-global-config-01-foundation.md` -> `0029-phase5-global-config-02-project-target-matching.md` -> `0030-phase5-global-config-03-materialization-profiles.md`

## Objective

Add simple project matching on top of the Phase 5 global config base. One user config should be able to pick different settings for different repos.

## Context

- `docs/project-plan.md` defines FR-28 as global project matching using the main worktree as the anchor.
- The current config model is repo-scoped only.
- Matching should stay simple for both humans and agents. Hidden fallback rules would work against that.

## Options and Trade-offs

1. **Use ordered project blocks with first-match-wins at the main worktree root (recommended)**
   - Pros: Deterministic and easy to read.
   - Cons: Large configs may require careful ordering by the user.
2. **Use automatic most-specific-match-wins**
   - Pros: Can reduce ordering burden.
   - Cons: Harder to explain when two patterns overlap.
3. **Use single exact-path match only**
   - Pros: Very simple.
   - Cons: Too rigid for the portability goal.

**Recommended option** Match project blocks in declared order against the main worktree root and stop at the first match.

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
| `docs/specs/configuration.md` | Add the project block format, match anchor, ordering rule, and unmatched behavior |
| `docs/specs/git-operations.md` | Clarify that target matching uses the repository main worktree root |
| `docs/project-plan.md` | Mark FR-28 status only if this child plan completes the full matching contract |

## Design Decision Changes

| File | Change |
|------|--------|
| `docs/design-decisions/adr.md` | Add an ADR for the match anchor and selection rule if needed |

## Code Changes

| File | Change |
|------|--------|
| `internal/config/config.go` | Add project blocks to global config selection |
| `internal/config/config_test.go` | Add target-matching tests, including overlapping and unmatched cases |
| `cmd/ww/main.go` | Ensure the matched project layer resolves before the manager config is finalized |

## Test Changes

- Add tests proving:
  - matching uses the main worktree root as the stable anchor
  - first matching project block wins
  - unmatched blocks leave the base global config unchanged
  - invalid project-target definitions fail clearly instead of half-applying

## Design Notes

- The target anchor must stay stable across `ww create`, `ww cd`, and runs from secondary worktrees.
- Skip implicit multi-match merging in this phase. One selected project block is easier to explain and debug.
- Keep the first matcher useful for path-based reuse.

## Sub-tasks

- [ ] [parallel] Specify the project-target syntax and first-match-wins rule in `docs/specs/configuration.md`
- [ ] [parallel] Decide whether `docs/specs/git-operations.md` needs an explicit anchor clarification
- [ ] [depends on: 0028 foundation] Add project-target parsing and selection in `internal/config`
- [ ] [depends on: implementation] Thread selected project settings into final manager config resolution
- [ ] [depends on: implementation] Add regression tests for overlapping rules, unmatched targets, and secondary-worktree invocation

## Verification

- `go test ./internal/config ./cmd/ww`
- Manual verification:
  - one global config applies different settings to two repos in the same workspace
  - invoking `ww` from a secondary worktree still matches the intended project by using the main repo root
  - overlapping project rules select the first declared match consistently
