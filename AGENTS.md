# ww — Workspace Worktree Manager

## Build & Test

```bash
make build    # Build binary with version info
make test     # Run only unit tests
make test-all # Run all tests (unit + integration)
make fmt      # Run formatter, use this instread of "go fmt"
make lint     # Run all linters
make clean    # Remove built binary
```

Integration tests build the `ww` binary and run it against temporary git repos. They live in `integration_test.go` at the repo root.

## Project Structure

```
cmd/ww/           # CLI entry point and subcommand wiring
git/              # Public: thin wrapper around git CLI (os/exec)
worktree/         # Public: business logic (create/list/remove)
validate/         # Public: branch name and path validation
internal/config/  # Internal: .ww.toml loader
docs/specs/       # Behavioral specs (no implementation details)
docs/exec-plan/   # Execution plans (todo/ and done/)
docs/issues/      # Known issues
```

**Public vs internal:** `git/`, `worktree/`, `validate/` are importable as a library. `internal/config/` is ww-specific. See [#5](https://github.com/yoskeoka/ww/issues/5) for a known coupling issue.

## Key Design Decisions

- **Git CLI, not library**: All git operations shell out to `git` via `os/exec` for maximum compatibility.
- **Main worktree resolution**: `ww` always resolves back to the main working tree using `git rev-parse --path-format=absolute --git-common-dir`. All path computations use the main repo, never the current worktree. This means `ww` works identically from any worktree.
- **POSIX-style flags**: Uses `--flag` syntax. Specs describe behavior only, not the flag parsing library.
- **Worktree path convention**: `<repo>@<sanitized-branch>` where `/` in branch names becomes `-`.

## Workflow

This project follows the AI-Centered Development workflow defined in the parent workspace:

1. **Spec first**: Update `docs/specs/` before changing code.
2. **Plan first**: Non-trivial changes need an execution plan in `docs/exec-plan/todo/`.
3. **Log issues**: Unrelated problems found during work go in `docs/issues/`.
4. **PR for everything**: All changes go through GitHub PR review.

## GitHub Actions Pinning

- When editing ordinary GitHub Actions workflows or composite actions, use `pinact` to pin or update `uses:` references rather than hand-editing version tags.

## Lessons Learned

- **Always pull before pushing on CI-active branches**: Agentic workflows may push commits (e.g., automated fixes) between your commits. Always `git pull --rebase` before pushing to avoid rejected pushes and merge conflicts.
