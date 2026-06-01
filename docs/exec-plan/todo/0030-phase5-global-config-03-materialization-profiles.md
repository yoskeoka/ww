# 0030: Phase 5 profile reuse

> **Execution**
> Run `/execute-task` for this plan.

**Parent plan series**
`0028-phase5-global-config-01-foundation.md` -> `0029-phase5-global-config-02-project-target-matching.md` -> `0030-phase5-global-config-03-materialization-profiles.md`

## Objective

Add reusable profiles on top of the Phase 5 global config base. Users should be able to define common worktree setup once and reuse it across repos.

## Context

- `docs/project-plan.md` defines FR-29 as hook-driven profiles for sandbox-friendly setup patterns.
- After Phase 5-01 and Phase 5-02, `ww` will have layered config and deterministic project selection.
- Profiles touch hooks and file/link setup, so the design must keep the agreed full-override model.

## Options and Trade-offs

1. **Use one named profile per project with full expansion (recommended)**
   - Pros: Keeps the runtime model simple.
   - Cons: Users cannot stack multiple profiles implicitly.
2. **Combine multiple profiles in order**
   - Pros: More flexible for advanced reuse.
   - Cons: Reintroduces the same merge and append ambiguity avoided in Phase 5-01.
3. **Skip profiles and repeat per-project config**
   - Pros: Smallest implementation.
   - Cons: Misses the portability goal.

**Recommended option** Allow one explicit profile selection that expands to normal config values.

## Scope

### In Scope

- Define the reusable profile format and where it can be referenced
- Specify how a selected profile expands into ordinary config fields
- Keep hook/copy/symlink behavior consistent with the full-override rule
- Add tests for profile selection, expansion, and invalid references

### Out of Scope

- Multi-profile stacking or additive profile composition
- Hook trust hardening beyond the existing trusted-config model
- New hook lifecycle phases; those belong to Phase 6

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/configuration.md` | Add named profile syntax, reference rules, expansion rules, and invalid-reference behavior |
| `docs/specs/cli-commands.md` | Document any CLI-visible output or diagnostics affected by profile expansion |
| `docs/project-plan.md` | Mark FR-29 status only if this child plan fully lands profile support |

## Design Decision Changes

| File | Change |
|------|--------|
| `docs/design-decisions/adr.md` | Add an ADR if needed to record why one profile was chosen over stacked composition |

## Code Changes

| File | Change |
|------|--------|
| `internal/config/config.go` | Add named materialization profiles to config resolution |
| `internal/config/config_test.go` | Add profile expansion and invalid-reference tests |
| `cmd/ww/main.go` | Use the fully resolved config produced by profile expansion without changing command-level behavior |
| `worktree/worktree.go` | Consume the resolved copy/symlink/hook values without new runtime policy branching |

## Test Changes

- Add tests proving:
  - a selected profile expands into ordinary config values before create/list/remove behavior runs
  - invalid profile references fail clearly
  - profile-provided arrays and hooks still obey full replacement semantics
  - repositories without profiles behave exactly as before

## Design Notes

- Profiles should stay as resolution-time sugar.
- Preserve the current trust boundary: if a selected profile defines a hook, that hook is still trusted config authored by the user.
- Skip multiple-profile composition in this phase. If users eventually need composition, add it as an explicit design.

## Sub-tasks

- [ ] [parallel] Specify named profile syntax and expansion semantics in `docs/specs/configuration.md`
- [ ] [depends on: 0028 foundation and 0029 target matching] Add profile parsing and single-profile expansion in `internal/config`
- [ ] [depends on: implementation] Ensure command setup consumes only fully resolved config values
- [ ] [depends on: implementation] Add regression tests for valid expansion, invalid references, and no-profile compatibility
- [ ] [depends on: verification] Update `docs/project-plan.md` status markers if FR-29 is fully complete

## Verification

- `go test ./internal/config ./cmd/ww ./worktree`
- Manual verification:
  - one global profile can be reused by multiple project-target blocks
  - the resolved create behavior matches the expanded `copy_files`, `symlink_files`, and `post_create_hook` values
  - invalid profile references fail with an actionable error
