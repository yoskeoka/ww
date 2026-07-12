# Workspace detection algorithm: parent-scan strategy

## Status

Superseded on 2026-03-31 by [0009](0009-workspace-detection-anchor.md).

## Context

Phase 2 adds workspace awareness — detecting when `ww` is inside a meta-repo with multiple child git repositories. Several detection strategies were considered:

- **Config-only**: Require explicit `workspace = true` in `.ww.toml`. Simple but violates "convention over configuration" — new users must configure before workspace features work.
- **Recursive scan**: Walk up the directory tree and scan all children at each level. Correct but slow and risks false positives on deeply nested repos.
- **Parent-scan (chosen)**: Check only the immediate parent directory for git siblings and the current directory for git children. Limited to one level in each direction.

## Decision

Parent-scan strategy with a 7-step algorithm:
1. Scan CWD children for `.git` entries (parent candidate)
2. Determine current git repo root
3. Check parent directory for `.git`
4. Check parent's children for `.git` siblings
5. (Reserved for future config override)
6. Fall back to CWD as workspace root if step 0 found children
7. None → single-repo mode

## Consequences

- **Positive**: Fast (only two directory scans), predictable, zero-config for standard meta-repo layouts.
- **Negative**: Cannot detect workspaces more than one level deep. Accepted — FR-22 (recursive detection) is reserved for the future if needed.
