# Exec-plan README Numeric Filename Staleness
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Resolve `docs/issues/exec-plan-readme-numeric-filename-stale.md` by bringing the exec-plan directory README guidance back in line with the current workflow.

The active README still documents numeric plan filenames such as `XXX-description.md`, but the current workspace workflow requires non-numeric, descriptive filenames that match the branch description. This plan keeps the fix limited to documentation and issue tracking.

## Context Read

- `docs/project-plan.md`: `ww` is intended to be agent-friendly and predictable. Stale workflow docs make automation and review feedback less predictable.
- `docs/design-decisions/core-beliefs.md`: spec-code parity and AI-first context retrieval matter more than preserving outdated navigation conventions.
- `docs/design-decisions/adr.md`: no existing ADR makes a product architecture decision about execution-plan naming. No ADR update is expected for this documentation cleanup.
- `docs/exec-plan/done/`: completed plans include a mix of historical numeric and newer descriptive names. That history should be preserved; the current guidance should describe future/current workflow behavior without renaming old plans.

## Code Changes

No production or test code changes are expected.

During execution, do not rename existing historical plan files in `docs/exec-plan/done/`. The issue explicitly scopes out that migration.

## Spec Changes

No `docs/specs/` change is expected because this is a workflow documentation cleanup, not a CLI behavior change.

During execution, search the docs for other stale execution-plan naming guidance. Update only directly related workflow README text if it repeats the obsolete numeric convention.

## Documentation and Issue Changes

- Update `docs/exec-plan/todo/README.md` to state:
  - active execution plans use non-numeric, descriptive kebab-case filenames
  - the filename matches the branch description, for example `plan/remove-docker-from-integration-tests` maps to `docs/exec-plan/todo/remove-docker-from-integration-tests.md`
  - completed plans move to `docs/exec-plan/done/` without being renamed into a numeric convention
- Update `docs/exec-plan/done/README.md` if useful so the archive guidance does not imply a separate numeric naming rule.
- Move `docs/issues/exec-plan-readme-numeric-filename-stale.md` to `docs/issues/done/exec-plan-readme-numeric-filename-stale.md` after the README guidance is corrected.

## Scope Options

- **Option A: Update only `docs/exec-plan/todo/README.md`.** This fixes the directly stale line with the smallest diff, but leaves the archive README too terse to prevent future confusion about whether completed plans need renaming.
- **Option B: Update both `docs/exec-plan/todo/README.md` and `docs/exec-plan/done/README.md`.** This is still a small documentation-only change and addresses the issue's related README scope.

Recommendation: choose Option B. It aligns both active and archive guidance while preserving historical completed filenames.

## Sub-tasks

- [x] [parallel] Search workflow docs for stale `XXX-description.md`, numeric prefix, or completed-plan renaming guidance.
- [x] [parallel] Update `docs/exec-plan/todo/README.md` with the current non-numeric naming convention.
- [x] [parallel] Update `docs/exec-plan/done/README.md` with archive behavior and no-renaming guidance.
- [x] [depends on: README updates] Move the active issue file into `docs/issues/done/`.
- [x] [depends on: README updates, issue move] Run documentation-focused verification and inspect the final diff.

## Parallelism

The stale-guidance search and the two README edits are independent. The issue file move should wait until the README updates are complete.

## Design Decisions

No new architecture decision is expected. This plan applies the existing workflow rule that execution-plan filenames are descriptive and non-numeric, matching the branch description.

## Verification

- `rg -n "XXX-description|001-init|numeric prefix|numbered|no numeric|branch description" docs/exec-plan docs/issues .github`
- `git diff --check`
- `git status --short`

## Success Criteria

- `docs/exec-plan/todo/README.md` no longer instructs agents to use numeric plan filenames.
- `docs/exec-plan/done/README.md`, if changed, clarifies archive behavior without requiring completed plans to be renamed.
- `docs/issues/exec-plan-readme-numeric-filename-stale.md` is moved to `docs/issues/done/`.
- No historical completed execution plans are renamed.
