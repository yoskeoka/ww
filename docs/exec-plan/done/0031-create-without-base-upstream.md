# Create Without Base Upstream

**Execution**: direct small bug fix

## Objective

Prevent `ww create <branch>` from leaving a newly created local branch tracking
the base ref used only for branch creation, such as `origin/main`.

Addresses: https://github.com/yoskeoka/ww/issues/260

## Spec Changes

- clarify in `docs/specs/cli-commands.md` that normal new-branch creation must
  not leave the new branch tracking the base branch
- document the follow-up `git branch --unset-upstream <branch>` behavior in
  `docs/specs/git-operations.md`

## Code Changes

- add a `git.Runner` helper that removes upstream only when one exists
- call that helper after `git worktree add -b ... <base>` in the normal
  new-branch creation path
- leave existing-branch and `--guess-remote` paths unchanged

## Verification

- `go test ./...`
- `make test`
- `make lint`
