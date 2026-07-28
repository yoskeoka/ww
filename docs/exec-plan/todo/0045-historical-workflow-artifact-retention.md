# Historical workflow artifact retention

> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

## Objective

Remove completed plan and local-issue bodies from the checked-out repository to
make normal code/task discovery active-only. Preserve auditability through the
plan PR, implementation PR, and Git history rather than `done/` directories or
long commit messages.

## Change map

- (MODIFY) `AGENTS.md`, `docs/exec-plan/todo/README.md`, `docs/issues/README.md`, `.github/copilot-instructions.md`, and all current workflow guidance that refers to `done/`.
- (MODIFY) `tools/workflow-lint.sh` and its focused test harness. Retain matching active-plan enforcement. Replace `todo`→`done` rename detection with a closeout-diff rule that reads the deleted plan from the merge-base side, requires deletion of explicit linked local issues, and retains external GitHub closing-keyword checks.
- (DELETE) all files under `docs/exec-plan/done/` and `docs/issues/done/`, then remove empty directories.

## Execution order

1. Specify the deletion-based lifecycle in the workflow-linter contract before code.
2. Implement and test separate active-plan and deleted-plan closeout paths.
3. Update guidance/templates, remove historical artifacts, and sweep stale references without rewriting ADRs or Git history.

## Verification

Run focused linter cases for active plan, valid deletion, omitted linked issue, and external issue metadata; then run `make test`, `make lint`, `git diff --check`, workflow-linter checks, and prove retrieval with `git log --all -- docs/exec-plan`.
