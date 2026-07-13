# Worktree STATUS: merged > stale, no-tracking = active

## Status

Accepted on 2026-03-18.

## Context

`ww list` adds a STATUS column (`active`/`merged`/`stale`). A branch could theoretically be both merged and stale (merged into base AND remote branch deleted). Branches without remote tracking could be considered stale by absence, but this would flag local-only feature branches incorrectly.

## Decision

- `merged` takes precedence over `stale` (if both conditions are met).
- Branches with no remote tracking configured are always `active`, never `stale`.
- Main worktrees are always `active`.

## Consequences

- **Positive**: Conservative — only flags worktrees as cleanable when there's strong evidence. Avoids false positives on local-only branches.
- **Negative**: A branch that was never pushed will stay `active` even if abandoned. Accepted — FR-23 (time-based stale detection) can address this later.
