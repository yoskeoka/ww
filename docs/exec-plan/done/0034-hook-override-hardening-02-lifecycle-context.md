# 0034: Hook Override Hardening 02 Lifecycle Context
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

**Shared branch**: `plan/hook-override-hardening`

## Objective

Expand the current post-create-only hook model into an explicit lifecycle surface that covers `pre-create`, `post-create`, `pre-remove`, and `post-remove`, and inject the missing hook context variables `WW_REPO_NAME` and `WW_WORKTREE_INDEX`. Completion means create/remove flows run the documented hook phases in a stable order, all shipped hook phases receive the same context contract, and specs describe the lifecycle precisely enough for repo-local automation to rely on it.

## Existing Implementation References

- `internal/config/config.go`
  - `Config`, lines 15-23
  - `configLayer`, lines 75-83
  - `configFields`, lines 85-92
  - `decodeFields`, lines 127-154
  - `mergeConfig`, lines 156-178
- `cmd/ww/main.go`
  - manager config mapping, lines 176-185
- `worktree/worktree.go`
  - create flow materialization order, lines 204-229
  - remove flow, lines 586-641
  - copy/symlink helpers, lines 658-688
  - `runPostCreateHook`, lines 690-713
- `docs/specs/configuration.md`
  - config field table and trust model, lines 50-82
- `docs/specs/cli-commands.md`
  - `ww create` hook behavior, lines 86-134
- `internal/config/config_test.go`
  - file-format and overlay coverage, lines 34-68 and 280-376

## Code Change Map

- `internal/config/config.go` (MODIFY)
  - extend config schema for additional lifecycle hooks and keep per-key replacement behavior explicit
- `cmd/ww/main.go` (MODIFY)
  - pass the expanded hook configuration into `worktree.Manager`
- `worktree/worktree.go` (MODIFY)
  - replace the ad hoc post-create runner with a lifecycle-aware hook executor, define phase ordering around create/remove, and inject the full environment contract including `WW_REPO_NAME` and `WW_WORKTREE_INDEX`
- `worktree/worktree_test.go` (MODIFY)
  - add create/remove hook ordering, env injection, and warning-path coverage
- `internal/config/config_test.go` (MODIFY)
  - add parsing and override tests for the new hook keys and empty-string replacement cases

## Spec Changes

- `docs/specs/configuration.md`
  - document the new lifecycle hook keys, their replacement behavior, and the full hook environment variable contract
- `docs/specs/cli-commands.md`
  - document create/remove lifecycle ordering, hook warnings, and which phases run on dry-run vs real execution

## Sub-tasks

- [ ] Extend config parsing and overlay behavior for `pre-create`, `post-create`, `pre-remove`, and `post-remove` hook fields
- [ ] Refactor hook execution so create/remove paths call shared lifecycle helpers with one consistent environment contract
- [ ] Add `WW_REPO_NAME` and `WW_WORKTREE_INDEX` to every real hook execution path and cover them with tests
- [ ] Define and verify failure behavior:
  - lifecycle hook failures warn to `stderr`
  - create/remove still follow the documented stop/continue rules for each phase
- [ ] Update hook lifecycle specs to match the implemented phase order and env surface

## Design Decisions

- Keep hook execution in the `worktree` layer, near the actual create/remove side effects, so phase ordering remains tied to the real lifecycle instead of CLI wrappers.
- Preserve the trusted-config model already documented for `post_create_hook`; this plan expands phases and context, not hook sandboxing or trust hardening.

## Parallelism

- This plan can execute in parallel with `0033-hook-override-hardening-01-child-repo-overrides.md`.
- `0035-hook-override-hardening-03-sandbox-replay.md` depends on this plan because replay should reuse the same lifecycle executor and phase naming.
