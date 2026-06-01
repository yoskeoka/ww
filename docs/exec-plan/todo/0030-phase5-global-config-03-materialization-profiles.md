# 0030: Phase 5 Global Config Materialization Profiles

> **Execution**: Use `/execute-task` to implement this plan.

**Parent plan series**: `0028-phase5-global-config-01-foundation.md` -> `0029-phase5-global-config-02-project-target-matching.md` -> `0030-phase5-global-config-03-materialization-profiles.md`

## Objective

Add reusable materialization profiles on top of the Phase 5 global-config foundation so users can centrally define common worktree setup patterns such as copy/symlink selections and hook-driven setup flows without committing the same `.ww.toml` payload into every repository.

## Context

- `docs/project-plan.md` defines FR-29 as hook-driven materialization profiles for sandbox-friendly setup patterns.
- After Phase 5-01 and Phase 5-02, `ww` will have a layered config source and deterministic project selection. This child plan uses that foundation to reduce repetition in real worktree-setup workflows.
- Materialization profiles interact directly with hooks and file/link setup, so the design must preserve the agreed full-override model and avoid hidden composition.

## Options and Trade-offs

1. **Named single-profile selection per project with explicit full expansion (recommended)**
   - Pros: Keeps the runtime model simple. One chosen profile expands to ordinary config values before command execution.
   - Cons: Users cannot stack multiple profiles implicitly.
2. **Multiple profiles combined in order**
   - Pros: More flexible for advanced reuse.
   - Cons: Reintroduces the same merge/append ambiguity Phase 5-01 deliberately avoided.
3. **No profile concept, only repeated per-project config**
   - Pros: Smallest implementation.
   - Cons: Misses the stated portability goal and keeps global config noisy and repetitive.

**Recommendation**: Option 1. Allow one explicit profile selection that expands to normal config values, with the existing complete-override semantics still applying at the final selected layer.

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
| `docs/specs/configuration.md` | Add named profile syntax, reference rules, expansion semantics, and invalid-reference behavior |
| `docs/specs/cli-commands.md` | Document any CLI-visible output or diagnostics affected by profile expansion |
| `docs/project-plan.md` | Mark FR-29 status only if this child plan fully lands profile support |

## Design Decision Changes

| File | Change |
|------|--------|
| `docs/design-decisions/adr.md` | Append ADR if needed to record why single-profile expansion was chosen over stacked profile composition |

## Code Changes

| File | Change |
|------|--------|
| `internal/config/config.go` | Extend config schema and resolution for named materialization profiles |
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

- Profiles should be resolution-time sugar, not a second execution policy layer inside `worktree.Manager`.
- Preserve the current trust boundary: if a selected profile defines a hook, that hook is still trusted config authored by the user.
- Do not combine multiple profiles in this phase. If users eventually need composition, it should be explicit and separately designed.

## Sub-tasks

- [ ] [parallel] Specify named profile syntax and expansion semantics in `docs/specs/configuration.md`
- [ ] [depends on: 0028 foundation and 0029 target matching] Implement profile parsing and single-profile expansion in `internal/config`
- [ ] [depends on: implementation] Ensure command setup consumes only fully resolved config values
- [ ] [depends on: implementation] Add regression tests for valid expansion, invalid references, and no-profile compatibility
- [ ] [depends on: verification] Update `docs/project-plan.md` status markers if FR-29 is fully complete

## Verification

- `go test ./internal/config ./cmd/ww ./worktree`
- Manual verification:
  - one global profile can be reused by multiple project-target blocks
  - the resolved create behavior matches the expanded `copy_files`, `symlink_files`, and `post_create_hook` values
  - invalid profile references fail with an actionable error
