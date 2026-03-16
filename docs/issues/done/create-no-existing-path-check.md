# ww create doesn't check for existing worktree at target path

**GitHub:** https://github.com/yoskeoka/ww/issues/6
**Type:** bug | **Priority:** Low

## Problem

If a worktree directory already exists at the computed path, `ww create` passes through to `git worktree add` which fails with a confusing git error. Should check upfront and give a clear message.

**File:** `worktree/worktree.go:Create()` — no pre-existence check before calling `WorktreeAdd`

## Proposed Solution

Before calling `git worktree add`, check if the target path already exists. If it does, return: `worktree already exists at <path>`. Symmetric with `ww remove` which already checks for existence.
