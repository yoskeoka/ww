# 016: Unknown Status When Default Base Cannot Be Confirmed

> **Execution**: Use `/execute-task` to implement this plan.

**Objective**: Make `ww list` and `ww clean` robust when a detected repository lacks a confirmed default base. Instead of failing the command, `ww` should continue listing worktrees, mark uncertain cases as `unknown(...)`, and exclude uncertain worktrees from cleanup.

**Issue**: `docs/issues/unknown-status-default-base.md`

## Background

Current status classification assumes every repository can resolve a base branch from:

- `default_base`
- `refs/remotes/origin/HEAD`

That assumption is too strict for workspace mode and for repositories with incomplete remote metadata. A repository may still expose strong hints such as `origin/main` or local branches tracking `origin/main`, but today `ww list` stops before returning any result.

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/cli-commands.md` | Add `unknown` status, define `status_detail`, document text rendering as `unknown(<detail>)`, and clarify that `--cleanable` / `ww clean` only act on `merged` and `stale` |
| `docs/specs/git-operations.md` | Document default-base resolution order, authoritative vs heuristic sources, and degraded behavior when `origin/HEAD` is absent |
| `docs/specs/configuration.md` | Clarify that explicit `default_base` remains authoritative |
| `docs/spec-code-mapping.md` | Update mappings for the status/default-base behavior if needed |

## Status Model

Status values become:

- `active`
- `merged`
- `stale`
- `unknown`

Add `status_detail` for degraded classification. Initial values:

- `no-origin-head`
- `heuristic-base`
- `no-remote`
- `remote-query-failed`
- `base-detect-failed`

Text output should render degraded cases as `unknown(<detail>)`.

## Default Base Resolution Order

When `default_base` is unset, resolve in this order:

1. `origin/HEAD`
2. local `main` tracking `origin/main`
3. multiple local branches converging on upstream `origin/main`
4. existence of remote ref `origin/main`
5. otherwise unresolved

If step 2-4 succeeds, treat the result as heuristic rather than authoritative.

## Behavioral Rules

- Authoritative base sources:
  - explicit `default_base`
  - `origin/HEAD`
- Only authoritative base resolution may produce `merged` / `stale` / `active`
- Heuristic-only base resolution must degrade to `status=unknown`, `status_detail=heuristic-base`
- Missing branch remote must degrade to `status=unknown`, `status_detail=no-remote`
- Remote query failures must degrade to `status=unknown`, `status_detail=remote-query-failed`
- `ww list --cleanable` and `ww clean` must ignore all `unknown` worktrees

## Code Changes

| File | Change |
|------|--------|
| `git/git.go` | Replace the current single-step default-branch lookup with a richer resolver that reports base ref, source, and degraded reason |
| `worktree/worktree.go` | Extend `WorktreeInfo` with `StatusDetail`, degrade status to `unknown` instead of returning fatal errors for uncertain classification, and keep cleanup eligibility restricted to confirmed `merged` / `stale` |
| `cmd/ww/sub_list.go` | Render text status as `unknown(<detail>)` and emit `status_detail` in JSON |
| `cmd/ww/sub_clean.go` | Keep cleanable filtering restricted to `merged` / `stale` and ensure `unknown` is never removed |
| `git/git_test.go` | Add unit coverage for default-base resolution order and source classification |
| `worktree/worktree_test.go` | Add unit coverage for degraded status classification |
| `integration_test.go` | Add integration coverage for missing `origin/HEAD`, no tracking remote, and `unknown` exclusion from `--cleanable` / `clean` |

## Design Decision Notes

This plan preserves cleanup safety over aggressiveness:

- uncertain worktrees remain visible
- uncertain worktrees are never auto-cleaned
- best-effort base inference improves diagnostics without silently converting uncertain repos into cleanable ones

If the implementation reveals a broader status-model tradeoff, append the decision to `docs/design-decisions/adr.md`.

## Sub-tasks

- [ ] [parallel] Update specs for `unknown` / `status_detail` behavior
- [ ] [parallel] Implement richer default-base resolution with authoritative vs heuristic sources
- [ ] [depends on: default-base resolver] Update worktree status classification and `WorktreeInfo`
- [ ] [depends on: status classification] Update CLI text/JSON rendering
- [ ] [depends on: status classification] Add unit and integration tests for degraded status behavior
- [ ] [depends on: all above] Verify `ww list`, `ww list --cleanable`, and `ww clean` behavior in workspace mode

## Verification

- `make test`
- `make test-all`
- Manual reproduction of the current failure case
- Confirm that affected worktrees now appear as `unknown(<detail>)`
- Confirm that `ww list --cleanable` and `ww clean` ignore `unknown`
