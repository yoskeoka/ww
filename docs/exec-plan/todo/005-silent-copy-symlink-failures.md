# 005: Fix Silent Copy/Symlink Failures

**Issue:** [docs/issues/silent-copy-symlink-failures.md](../../issues/silent-copy-symlink-failures.md) | [GitHub #8](https://github.com/yoskeoka/ww/issues/8)
**Objective:** Distinguish missing-source errors (skip silently) from other failures (warn on stderr) in `copyFiles()` and `symlinkFiles()`.

## Context

Currently `copyFiles()` and `symlinkFiles()` in `worktree/worktree.go` silently ignore **all** errors. The spec says missing sources should be silently skipped, but other failure modes (permission denied, disk full, broken symlinks) are also swallowed. This makes debugging hard when worktree setup fails.

The fix follows the existing pattern from post-create hook error handling:
```go
fmt.Fprintf(os.Stderr, "warning: post-create hook failed: %v\n", err)
```

## Sub-tasks

- [ ] [parallel] **Spec update**: Create `docs/specs/worktree-file-operations.md` documenting copy/symlink error handling behavior (silent skip for `os.IsNotExist`, warning for other errors).
- [ ] [parallel] **Fix `copyFiles()`**: After `copyPath()` fails, check `os.IsNotExist(err)` — if true, `continue`; otherwise `fmt.Fprintf(os.Stderr, "warning: could not copy %s: %v\n", pattern, err)`.
- [ ] [parallel] **Fix `symlinkFiles()`**: Same pattern — silent skip for missing source, warn for `MkdirAll` and `Symlink` failures.
- [ ] [depends on: Fix copyFiles, Fix symlinkFiles] **Add unit tests**: Add tests to `worktree/worktree_test.go` covering:
  - Missing source → no warning, no error.
  - Permission denied → warning printed to stderr.
  - Successful copy/symlink → no warning.
- [ ] [depends on: Add unit tests] **Move issue to done**: Move `docs/issues/silent-copy-symlink-failures.md` → `docs/issues/done/`.

## Code Changes

| File | Change |
|------|--------|
| `worktree/worktree.go` | Update `copyFiles()` and `symlinkFiles()` error handling |
| `worktree/worktree_test.go` | Add test cases for error handling |

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/worktree-file-operations.md` | New — document copy/symlink error handling contract |

## Design Notes

- No new dependencies needed — uses only `os.IsNotExist()` and `fmt.Fprintf`.
- Warnings go to stderr (not stdout) to avoid breaking `--json` or shell integration output.
- Matches the existing warning pattern used by post-create hooks (line 262-264).
