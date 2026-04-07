# 017: Heuristic Base Branch Resolution

> **Execution**: Use `/execute-task` to implement this plan.

**Objective**: When `default_base` is unset and `origin/HEAD` is unavailable, attempt heuristic base branch resolution before falling back to `unknown` status. If a heuristic base is found, perform normal status classification (`merged`/`stale`/`active`) and annotate the result with `status_detail=heuristic-base`.

**Depends on**: `docs/exec-plan/done/016-unknown-status-default-base.md`

## Background

Plan 016 introduced graceful degradation when the base branch cannot be determined — worktrees receive `unknown(base-detect-failed)` status instead of the command failing. However, many repositories without `origin/HEAD` still have enough metadata to infer a base branch (e.g., a local `main` tracking `origin/main`, or a remote ref `origin/main` that exists).

The current implementation skips straight to `unknown` when authoritative sources fail. This plan adds heuristic steps that attempt to resolve a usable base before giving up.

## Heuristic Resolution Order

When `default_base` is unset and `origin/HEAD` fails:

1. Check if local `main` branch tracks `origin/main` (`git config --get branch.main.remote` == `origin`)
2. Check if any local branch tracks `origin/main` (scan `branch.*.remote` config)
3. Check if remote ref `origin/main` exists (`git ls-remote --heads origin main`); if not found, try `origin/master`
4. If none succeed → `unknown(base-detect-failed)` (current behavior)

If step 1-3 succeeds, use the resolved ref (e.g., `origin/main` or `origin/master`) as the base and classify statuses normally.

### Candidate branch names

Steps 1-3 try `main` first, then `master`. This covers the vast majority of repositories (GitHub defaults to `main`, older repos and GitLab often use `master`). Other default branch names (`develop`, `trunk`, etc.) are not tried — those projects should set `default_base` explicitly in `.ww.toml`.

This is an intentional simplification. The candidate list can be extended in the future if needed, but keeping it short avoids unnecessary `ls-remote` calls and keeps the heuristic predictable.

## Behavioral Rules

- Authoritative base sources (from 016):
  - explicit `default_base`
  - `origin/HEAD`
- Heuristic base resolution produces normal statuses (`merged`/`stale`/`active`) with `status_detail=heuristic-base`
- `ww list --cleanable` and `ww clean` should include heuristic-resolved `merged`/`stale` worktrees (they are safe to clean — the base was confirmed, just not via the authoritative channel)
- Only truly unresolvable repos get `unknown` status

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/git-operations.md` | Document the heuristic resolution steps and their order |
| `docs/specs/cli-commands.md` | Document `status_detail=heuristic-base` as a valid detail value |

## Code Changes

| File | Change |
|------|--------|
| `git/git.go` | Add `HeuristicDefaultBranch()` method that attempts the heuristic steps in order, returning the resolved ref and a bool indicating whether it's heuristic |
| `worktree/worktree.go` | Update `baseRef()` to return a struct with ref, source type (authoritative/heuristic/none); update `listRepo()` to set `status_detail=heuristic-base` on all entries when base was resolved heuristically |
| `git/git_test.go` | Unit tests for `HeuristicDefaultBranch()` |
| `worktree/worktree_test.go` | Unit tests for heuristic status detail propagation |
| `integration_test.go` | Integration test: repo with `origin` remote but no `origin/HEAD`, local `main` tracking `origin/main` → statuses should be `merged`/`stale`/`active` with `status_detail=heuristic-base` |

## Sub-tasks

- [x] Update specs for `heuristic-base` status detail
- [x] Implement `HeuristicDefaultBranch()` in `git/git.go`
- [x] Update `baseRef()` and `listRepo()` to use heuristic resolution
- [x] Add unit and integration tests
- [x] Verify `--cleanable` and `ww clean` include heuristic-resolved worktrees

## Verification

- `make test`
- `make test-all`
- Manual test with a repo that has `origin` but no `origin/HEAD`
- Confirm heuristic-resolved worktrees show `status_detail=heuristic-base`
- Confirm `--cleanable` includes heuristic `merged`/`stale` worktrees
