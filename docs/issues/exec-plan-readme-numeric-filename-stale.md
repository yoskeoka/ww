# Exec-plan README still documents numeric plan filenames

**Type:** documentation | **Priority:** Low

## Problem

`docs/exec-plan/todo/README.md` still says active execution plans should use `XXX-description.md` filenames. The current workspace workflow and `plan-execution` skill require non-numeric, descriptive filenames that match the branch description, for example:

- branch: `plan/remove-docker-from-integration-tests`
- plan: `docs/exec-plan/todo/remove-docker-from-integration-tests.md`

The stale README misleads reviewers and automation. Copilot flagged PR #172 for not using a numeric prefix because it relied on this outdated README.

## Expected Behavior

The README should match the canonical workflow:

- no numeric prefixes
- plan filename matches the branch description
- completed plans move from `docs/exec-plan/todo/` to `docs/exec-plan/done/` without renaming to a numbered convention

## Scope

Update `docs/exec-plan/todo/README.md` and any related README in `docs/exec-plan/done/` if present. Do not rename existing historical done plans unless a separate migration plan explicitly calls for it.
