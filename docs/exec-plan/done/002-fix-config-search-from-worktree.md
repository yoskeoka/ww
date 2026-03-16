# 002: Fix config search from worktree

## Objective

Fix the bug where `config.Load()` cannot find `.ww.toml` when `ww` is run from a sibling worktree (e.g., `repo@feat-x/`). Resolves [GH #4](https://github.com/yoskeoka/ww/issues/4).

**Problem**: `findConfig()` searches upward from CWD. When CWD is a sibling worktree like `repo@feat-x/`, the upward search goes to the parent directory but never checks inside the main repo directory where `.ww.toml` lives.

```
parent/
  repo/           <- .ww.toml is here
  repo@feat-x/    <- CWD, upward search goes to parent/ then /
```

**Solution**: After the upward search fails, fall back to checking the main worktree directory. The main worktree path can be resolved via `git rev-parse --path-format=absolute --git-common-dir` (strip trailing `/` and `.git`). This aligns with the existing design decision in CLAUDE.md: "ww always resolves back to the main working tree."

## Spec Changes

Update `docs/specs/configuration.md` section "Config Search":

- Add step between 4 and 5: "If not found via upward search and CWD is inside a git worktree, also check the main worktree's root directory."
- This makes config resolution worktree-aware without changing behavior for non-worktree usage.

## Code Changes

### `internal/config/config.go`

- [ ] Add `findConfigFromMainWorktree(startDir string) string` — resolves the main worktree path using git and checks for `.ww.toml` there.
- [ ] Update `findConfig()` (or `Load()`) to call the fallback when upward search returns empty.

### `internal/config/config_test.go`

- [ ] [parallel] Add unit test: config found from sibling worktree via main worktree fallback.
- [ ] [parallel] Add unit test: upward search still takes priority (config in parent dir wins over main worktree).
- [ ] [parallel] Add unit test: fallback gracefully returns empty when not in a git repo.

### `integration_test.go`

- [ ] Add integration test: create a worktree, place `.ww.toml` in main repo, run `ww list` from worktree, verify config is loaded.

## Sub-tasks

1. [ ] Update `docs/specs/configuration.md` with new config search step
2. [ ] [parallel] Implement `findConfigFromMainWorktree()` in `internal/config/config.go`
3. [ ] [parallel] Write unit tests in `internal/config/config_test.go`
4. [ ] [depends on: 2, 3] Wire fallback into `findConfig()` / `Load()`
5. [ ] [depends on: 4] Write integration test
6. [ ] [depends on: 5] Run `make test && make lint`, fix any failures

## Design Notes

- The git command approach (`git rev-parse --git-common-dir`) is consistent with the project's NFR-3 (use git CLI, not a Go git library).
- The fallback only activates when the upward search fails, preserving backward compatibility.
- No changes to the public `Config` struct or `Load()` signature are needed. The `startDir` parameter is sufficient context.
