# ww — Workspace Worktree Manager

## Build & Test

```bash
make build    # Build binary with version info
make test     # Run all tests (unit + integration)
make lint     # Run go vet
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

## Automated PR Review (gh-aw)

Three workflows automatically review PRs: `plan-review`, `impl-review`, `spec-code-sync`. They post Approve or Request Changes based on file path patterns.

**Handling false positives:**
- If a review requests changes incorrectly, bypass the rule to merge
- Log each false positive as a `docs/issues/` entry describing the trigger and why it was wrong
- Use logged false positives to refine the workflow prompts in `.github/aw/`
