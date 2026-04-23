# Consider Allowing Relative `worktree_dir` Paths Outside the Anchor

## Summary

`ww` currently rejects relative `worktree_dir` values that escape their anchor with `..`. This applies before git worktree creation and prevents configs such as:

```toml
worktree_dir = "../worktrees"
```

The rule is conservative, but it may be more surprising than helpful because absolute `worktree_dir` values are already accepted. A future task should evaluate whether `ww` should treat explicit `worktree_dir` as user intent regardless of whether it is absolute or relative.

## Context

The sandbox-constrained mode plan keeps the existing `..` escape rejection to avoid changing unrelated path-safety behavior in the same task. During planning, we identified a possible simplification:

- explicit `worktree_dir` is trusted config authored by the repo/user
- absolute paths can already point anywhere
- allowing relative paths to point outside the anchor would make absolute and relative override behavior consistent
- real filesystem sandbox denial would still surface as the underlying operation error

## Follow-up Question

Should `worktree_dir` accept any explicit path, absolute or relative, and only reject invalid paths for reasons like empty names, branch path traversal, control characters, or filesystem/git failures?

## Candidate Scope

- Update `docs/specs/configuration.md` and `docs/specs/workspace-discovery.md`.
- Revisit the relative path escape checks in `worktree/worktree.go`.
- Update tests that currently expect `../` escaping `worktree_dir` values to fail.
- Decide whether this should apply in all modes or only under an explicit compatibility flag.
