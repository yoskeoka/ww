# 004: Guard against removing the main worktree

## Objective

Harden `ww remove` by switching from filesystem checks to git as source of truth, and adding a main worktree guard. Resolves two issues:

- `docs/issues/remove-main-worktree-guard.md` — no guard against removing the main worktree
- `docs/issues/remove-uses-stat-not-git.md` — uses `os.Stat` instead of `git worktree list` for existence check

**Problem 1 (main worktree guard)**: `Remove()` never checks whether the target is the main worktree. If a user runs `ww remove main`, git refuses with a confusing error instead of a clear message.

**Problem 2 (stat vs git)**: `Remove()` checks `os.Stat(wtPath)` to verify the worktree exists. If the worktree directory was manually deleted but is still registered in git's worktree list, `ww remove` fails with "no worktree found" instead of cleaning up the stale registration.

**Solution**: Replace the `os.Stat` check with a lookup against `git worktree list --porcelain` output. Find the entry matching the given branch, reject if `Main == true`, and use git's recorded path for removal. This solves both issues in one change.

## Spec Changes

Update `docs/specs/cli-commands.md` section `ww remove <branch>`:

- Add step before "Verify the worktree exists": "Reject if the branch belongs to the main worktree: `cannot remove the main worktree`."
- Update "Verify the worktree exists" to clarify it checks `git worktree list`, not filesystem.

## Code Changes

### `worktree/worktree.go` — `Remove()`

- [ ] Replace `os.Stat(wtPath)` existence check with a lookup in `m.Git.WorktreeList()` output.
- [ ] If the matching entry has `Main == true`, return error: `cannot remove the main worktree`.
- [ ] If no matching entry found, return error: `no worktree found for branch "<branch>"`.
- [ ] Use the entry's `Path` (from git) instead of the computed `wtPath` for the `git worktree remove` call, ensuring consistency.

### `worktree/worktree_test.go`

- [ ] [parallel] Add unit test: Remove returns error when target is the main worktree.
- [ ] [parallel] Add unit test: Remove returns error when branch has no worktree entry.
- [ ] [parallel] Add unit test: Remove succeeds for a non-main worktree (happy path with mock).

## Sub-tasks

1. [ ] Update `docs/specs/cli-commands.md` with main worktree guard behavior
2. [ ] [parallel] Implement guard logic in `worktree/worktree.go:Remove()`
3. [ ] [parallel] Write unit tests in `worktree/worktree_test.go`
4. [ ] [depends on: 2, 3] Run `make test && make lint`, fix any failures
5. [ ] [depends on: 4] Verify with integration test if applicable
6. [ ] [depends on: 5] Move `docs/issues/remove-main-worktree-guard.md` and `docs/issues/remove-uses-stat-not-git.md` to `docs/issues/done/`

## Design Notes

- Using `git worktree list` as source of truth (instead of `os.Stat`) aligns with NFR-3 (use git CLI) and fully resolves the `remove-uses-stat-not-git` issue.
- The `Main` flag is already parsed by `git.Runner.WorktreeList()` so no new git plumbing is needed.
- Error message follows existing style: lowercase, no punctuation, descriptive.
