# 019: Tighten workspace repo detection and do not follow child symlinks by default

> **Execution**: Use `/execute-task` to implement this plan.

**Objective**: Make workspace discovery count only real child git repositories and exclude symlinked child entries by default. This prevents false workspace members such as `.claude` and `.agents` from appearing in `ww list` due to stray `.git` directories, while keeping future room for an explicit symlink-following mode if it becomes necessary.

## Background

The current workspace discovery code treats an immediate child as a repository when `<child>/.git` exists as either a directory or a regular file, with a special-case exclusion only for `.git/worktrees/...` file pointers. That heuristic is too loose:

- A child directory can contain a `.git` directory that is not a valid standalone git repository.
- A child directory can also expose shared content through symlinks without being intended as a workspace repository.
- In the observed workspace, `.claude` and `.agents` are listed because they contain `.git` directories, even though `git -C <dir> rev-parse --show-toplevel` resolves to the parent workspace repo rather than the child itself.
- `.gemini` is not listed because it has no `.git`, even though it shares the same `skills` symlink pattern.

The desired behavior is stricter:

- A workspace child counts only if it is itself a real git repository root.
- Child symlink entries are not followed by default.
- Managed git worktrees remain excluded from workspace-member detection.

This aligns with the existing design principle of correctness over speed: workspace discovery should prefer fewer false positives even if the check is slightly more expensive.

## Design Direction

Past decision: workspace detection is anchored on the current repo's main worktree root and only counts immediate child **real git repositories**. Apply the same reasoning here by tightening what "real git repository" means instead of broadening discovery heuristics.

For this plan, "real child git repository" should mean:

- the immediate child entry itself is not a symlink
- `git -C <child> rev-parse --show-toplevel` succeeds
- the resolved top-level path matches the child directory itself
- worktree checkouts that resolve elsewhere must not count as workspace members

Future-facing note:

- Do not add symlink-following behavior in this plan.
- Leave a clear seam for a future opt-in mode if there is a strong use case for following child symlinks under explicit rules.

## Spec Changes

### `docs/specs/workspace-discovery.md`

- Replace the current `.git`-marker wording with a stricter definition of "real git repository"
- State that immediate child symlink entries are ignored during workspace-member discovery
- Clarify that `.git` presence alone is insufficient; the child must resolve as its own git top-level
- Preserve the rule that managed git worktree checkouts are excluded
- Add an edge-case note covering false-positive `.git` directories and non-repo helper directories

### `docs/specs/cli-commands.md`

- Confirm that workspace-sensitive commands (`list`, `clean`, `create --repo`, `remove --repo`) only operate on repositories returned by the stricter discovery algorithm
- Update wording if any command description implies that `.git` markers alone define workspace membership

### `docs/specs/configuration.md`

- Check whether any configuration-search wording assumes that helper directories or symlinked children can become workspace members
- Update only if needed; avoid unrelated config changes

## Code Changes

### `workspace/workspace.go`

- Replace `hasGitEntry()`-style child classification with a stricter repository validation helper
- Ignore immediate child entries that are symlinks
- Validate candidate child repositories by asking git for the child's top-level path and comparing it to the child path
- Preserve graceful behavior when git is unavailable or a candidate is not a valid repository
- Keep the bounded-window and nearest-containing-workspace-root logic unchanged apart from the stricter child-repo test

### `git/git.go`

- Add a small helper if needed for "show me the repo top-level for this directory" so workspace discovery does not duplicate git-command glue
- Keep the helper narrowly scoped; avoid broad refactors

### `cmd/ww/main.go`

- Verify no behavior change is needed beyond consuming the updated workspace detection result
- Adjust only if stricter discovery exposes an assumption in manager creation

## Test Changes

### `workspace/workspace_test.go`

- Add unit test: child with a stray `.git` directory but no valid repo structure is ignored
- Add unit test: child symlink pointing at a real repository is ignored by default
- Add unit test: child with a `.git` file or directory that resolves to its own top-level still counts as a repository
- Keep or update existing tests that ensure git worktree checkouts are excluded

### `integration_test.go`

- Add integration test reproducing the false-positive case: helper directories with invalid `.git` contents do not appear in `ww list`
- Add integration test confirming a symlinked child repo is not treated as a workspace member
- Verify that normal child repositories in the same workspace are still listed

## Sub-tasks

- [ ] Update workspace discovery spec to define real child repositories and the default no-follow symlink policy
- [ ] Refine workspace child-repo detection to require a valid git top-level match
- [ ] [parallel] Add unit tests for invalid `.git` directories, valid child repos, and ignored child symlinks
- [ ] [parallel] Add integration coverage for false-positive helper directories and symlinked child entries
- [ ] [depends on: spec update, implementation, tests] Verify workspace-aware commands still operate correctly with the stricter repo set

## Verification

- `go test ./workspace`
- `go test ./git`
- `go test ./...`
- Manual verification in a workspace shaped like:
  - normal child repo directories
  - helper directories containing stray `.git` directories
  - child directories containing symlinks into another repo
- Confirm that `ww list` includes only true child repositories and excludes helper directories such as `.claude` / `.agents`

## Expected Outcome

- Workspace discovery no longer treats arbitrary `.git` markers as sufficient proof of a child repository
- `ww list` reflects only real workspace repositories
- Child symlink entries are ignored by default, avoiding ambiguity until an explicit opt-in policy exists
