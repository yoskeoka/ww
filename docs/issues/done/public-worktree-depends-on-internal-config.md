# Public worktree package depends on internal/config

**GitHub:** https://github.com/yoskeoka/ww/issues/5
**Type:** enhancement | **Priority:** Medium

## Problem

`worktree.Manager` has a field `Config *config.Config`, but `config` is under `internal/`. External consumers can import `worktree` (it's a public package) but cannot construct a `Config` value to pass to `Manager`.

**File:** `worktree/worktree.go:18-22`

## Proposed Solution

Option A: Move config out of `internal/`.
Option B: Accept plain values in Manager (replace `Config *config.Config` with individual fields). The CLI layer handles config loading and maps fields. Option B is cleaner for library consumers.
