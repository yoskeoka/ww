# Config search from worktree doesn't find main repo's .ww.toml

**GitHub:** https://github.com/yoskeoka/ww/issues/4
**Type:** bug | **Priority:** High

## Problem

`config.Load()` searches upward from CWD for `.ww.toml`. When running `ww` from a sibling worktree (e.g., `repo@feat-x/`), the upward search goes to the parent directory but never checks inside the main repo directory.

Config placed at `repo/.ww.toml` is invisible from `repo@feat-x/`.

**File:** `internal/config/config.go:findConfig()`

## Proposed Solution

After the upward search fails, also check the main worktree directory (available via `git rev-parse --path-format=absolute --git-common-dir`). Either pass the main worktree path into `config.Load()`, or accept a list of fallback directories.
