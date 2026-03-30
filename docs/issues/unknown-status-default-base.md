# `ww list` fails hard when default base cannot be confirmed

## Summary

`ww list` currently aborts for the entire detected repo/workspace when one detected repository does not expose a confirmed default base via `refs/remotes/origin/HEAD`.

This is too strict. A repository may be healthy and usable while still missing `origin/HEAD`. In that case, `ww` should still list worktrees and degrade status classification instead of terminating the command.

## Reproduction

Example failure:

```text
./ww list
resolving base branch for reversi-adventure: cannot detect default branch: git symbolic-ref refs/remotes/origin/HEAD: exit status 128
fatal: ref refs/remotes/origin/HEAD is not a symbolic ref
```

## Cause Analysis

- `ww list` computes a `STATUS` for each worktree instead of only printing `git worktree list`.
- The current `merged` / `stale` classification depends on resolving a base branch and running `git branch --merged <base>`.
- Base resolution currently requires either:
  - explicit `default_base` config, or
  - `git symbolic-ref refs/remotes/origin/HEAD`
- If that lookup fails for any detected repository, `listRepo()` returns an error and the whole command stops.

## Why This Is A Problem

- Missing `origin/HEAD` is not inherently a broken Git state.
- Workspace-wide visibility becomes fragile: one repo with incomplete remote metadata can block `ww list`.
- `ww clean` depends on the same status pipeline and can be blocked for the same reason.

## Expected Behavior

- `ww list` should still return worktrees when base detection is incomplete.
- Status should degrade to `unknown(...)` instead of hard-failing.
- `ww list --cleanable` and `ww clean` should remain limited to confirmed `merged` and `stale` worktrees.

## Planned Resolution

Address this with `docs/exec-plan/todo/016-unknown-status-default-base.md`:

- add `status=unknown`
- add `status_detail`
- infer likely base refs in a best-effort sequence
- avoid treating uncertain worktrees as cleanable
