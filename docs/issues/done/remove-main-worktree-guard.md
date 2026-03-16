# ww remove doesn't guard against removing the main worktree

**GitHub:** https://github.com/yoskeoka/ww/issues/7
**Type:** bug | **Priority:** Medium

## Problem

Running `ww remove main` (or whatever the main branch is) would attempt to remove the main working tree. Git will refuse, but the error message is confusing.

**File:** `worktree/worktree.go:Remove()` — no check against removing the main worktree

## Proposed Solution

Before removing, compare the target path against `Manager.RepoDir`. If they match, return: `cannot remove the main worktree`.
