# Silent failures in copy/symlink operations

**GitHub:** https://github.com/yoskeoka/ww/issues/8
**Type:** enhancement | **Priority:** Low

## Problem

`copyFiles()` and `symlinkFiles()` silently ignore all errors. The spec says "missing sources are silently skipped", but other failure modes (permission denied, disk full) are also silently swallowed.

**File:** `worktree/worktree.go:210-232`

## Proposed Solution

- Skip silently when source doesn't exist (`os.IsNotExist`).
- For other errors, print a warning to stderr: `warning: could not copy <file>: <error>`.
- Matches the pattern already used for post-create hook failures (line 248).
