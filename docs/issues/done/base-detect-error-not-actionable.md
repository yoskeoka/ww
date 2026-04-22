# Base branch detection error is not actionable

## Summary

When `ww create` fails because the base branch cannot be determined, the error message exposes raw git internals and gives no guidance on how to fix it.

## Current behavior

```
cannot determine base branch: cannot detect default branch: git symbolic-ref refs/remotes/origin/HEAD: exit status 128
fatal: ref refs/remotes/origin/HEAD is not a symbolic ref
```

## Expected behavior

```
cannot determine base branch: origin/HEAD is not set.
Set default_base in .ww.toml or run: git remote set-head origin --auto
```

The error should:
1. Explain what's missing in plain language
2. Provide actionable remediation steps

## Affected commands

- `ww create` — fails with the raw error when neither `default_base` nor `origin/HEAD` is available
- `ww list` / `ww clean` — no longer fail (016 added graceful degradation), but the underlying `DefaultBranch()` error message is still poor if logged

## Fix

Improve the error message in `git.Runner.DefaultBranch()` or in `worktree.Manager.Create()` to include remediation hints.
