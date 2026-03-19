# 005: Fix Silent Copy/Symlink Failures

**Issue:** [docs/issues/silent-copy-symlink-failures.md](../../issues/silent-copy-symlink-failures.md) | [GitHub #8](https://github.com/yoskeoka/ww/issues/8)
**Objective:** Distinguish missing-source errors (skip silently) from other failures (warn on stderr) in `copyFiles()` and `symlinkFiles()`.

## Context

Currently `copyFiles()` and `symlinkFiles()` in `worktree/worktree.go` silently ignore **all** errors. The spec says missing sources should be silently skipped, but other failure modes (permission denied, disk full, broken symlinks) are also swallowed. This makes debugging hard when worktree setup fails.

The fix follows the existing pattern from post-create hook error handling:
```go
fmt.Fprintf(os.Stderr, "warning: post-create hook failed: %v\n", err)
```

### Broken symlink handling

Use `os.Lstat` (not `os.Stat`) to check source existence. `os.Stat` follows symlinks, so a broken symlink returns `IsNotExist` and would be silently skipped — hiding a real problem. With `os.Lstat`:
1. `os.Lstat(src)` returns `IsNotExist` → source truly doesn't exist → skip silently.
2. `os.Lstat(src)` succeeds but `os.Stat(src)` fails → source is a broken symlink → warn.
3. Both succeed → proceed normally.

## Sub-tasks

- [ ] [parallel] **Spec update**: Create `docs/specs/worktree-file-operations.md` documenting copy/symlink error handling behavior (silent skip for truly missing sources, warn for broken symlinks and other errors). Use `os.Lstat`/`os.Stat` combo for source checks.
- [ ] [depends on: Spec update] **Update spec-code mapping**: Add a row to `docs/spec-code-mapping.md` mapping `docs/specs/worktree-file-operations.md` to `worktree/` and `worktree/worktree_test.go`.
- [ ] [parallel] **Fix `copyFiles()`**: Use `os.Lstat` to check source. If `os.IsNotExist` → `continue`. If Lstat succeeds but Stat fails (broken symlink) → warn. After `copyPath()` fails for other reasons → warn. Pattern: `fmt.Fprintf(os.Stderr, "warning: could not copy %s: %v\n", pattern, err)`.
- [ ] [parallel] **Fix `symlinkFiles()`**: Same `os.Lstat`/`os.Stat` pattern for source check. Also handle `os.Stat` errors that aren't `os.IsNotExist` (e.g., permission denied on stat) → warn. Warn on `MkdirAll` and `Symlink` failures too.
- [ ] [depends on: Fix copyFiles, Fix symlinkFiles] **Add unit tests**: Add tests to `worktree/worktree_test.go` covering:
  - Missing source → no warning, no error.
  - Broken symlink source → warning printed to stderr.
  - Permission denied → warning printed to stderr.
  - Successful copy/symlink → no warning.
- [ ] [depends on: Add unit tests, Update spec-code mapping] **Move issue to done**: Move `docs/issues/silent-copy-symlink-failures.md` → `docs/issues/done/`.

## Code Changes

| File | Change |
|------|--------|
| `worktree/worktree.go` | Update `copyFiles()` and `symlinkFiles()` error handling |
| `worktree/worktree_test.go` | Add test cases for error handling |

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/worktree-file-operations.md` | New — document copy/symlink error handling contract |
| `docs/spec-code-mapping.md` | Add mapping row for `docs/specs/worktree-file-operations.md` → `worktree/`, `worktree/worktree_test.go` |

## Design Notes

- No new dependencies needed — uses only `os.IsNotExist()` and `fmt.Fprintf`.
- Warnings go to stderr (not stdout) to avoid breaking `--json` or shell integration output.
- Matches the existing warning pattern used by post-create hooks (line 262-264).
