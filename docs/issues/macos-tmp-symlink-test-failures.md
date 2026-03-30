# macOS /tmp symlink causes unit test failures

## Summary

`TestFindByName` and `TestMostRecentUsesWorktreeAdminMtime` fail on macOS because `/tmp` is a symlink to `/private/tmp`. Git resolves the real path while test expectations use the symlink path.

## Reproduction

```bash
make test
```

```
--- FAIL: TestFindByName (0.13s)
    worktree_test.go:247: FindByName returned path "/private/tmp/claude/...", want "/tmp/claude/..."
--- FAIL: TestMostRecentUsesWorktreeAdminMtime (0.25s)
    worktree_test.go:267: could not find admin dir for /tmp/claude/...
```

## Fix

Use `filepath.EvalSymlinks` or `os.MkdirTemp` with a non-symlink base path in the test setup helper, so that expected and actual paths match.
