# 004: Guard against removing the main worktree

## Objective

Prevent `ww remove` from attempting to remove the main working tree. Currently, if a user runs `ww remove main` (or whatever branch the main worktree is on), the command passes through to `git worktree remove` which fails with a confusing error. Add an explicit guard with a clear error message. Resolves `docs/issues/remove-main-worktree-guard.md`.

**Problem**: `Remove()` computes the worktree path via `WorktreePath(branch)` and checks `os.Stat`, but never compares the resolved path against `Manager.RepoDir`. The main worktree lives at `RepoDir`, not at the sibling path layout, so `os.Stat` would actually fail for the main branch (since there's no `repo@main` directory) — but if someone names a branch that collides, or if the path layout changes, this is still a gap. More importantly, we should check against the git worktree list to identify the main worktree by its `main` flag, not by path guessing.

**Solution**: In `Remove()`, before proceeding, list worktrees via `git worktree list --porcelain`, find the entry for the given branch, and reject if it is the main worktree. This also partially addresses the `remove-uses-stat-not-git` issue (using git as source of truth instead of `os.Stat`).

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

## Design Notes

- Using `git worktree list` as source of truth (instead of `os.Stat`) aligns with NFR-3 (use git CLI) and partially resolves the `remove-uses-stat-not-git` issue.
- The `Main` flag is already parsed by `git.Runner.WorktreeList()` so no new git plumbing is needed.
- Error message follows existing style: lowercase, no punctuation, descriptive.
