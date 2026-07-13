# Workspace detection anchor: nearest containing workspace within a bounded window (current dir + main root/parent/grandparent)

## Status

Accepted on 2026-03-31. Supersedes [0002](0002-workspace-detection-parent-scan.md).

## Context

Workspace detection originally aimed to stay local: reason from the current repository, look one level up for a containing workspace, and look one level down for child repositories. The current implementation drifted beyond that and added a grandparent guard: when the parent directory is itself a git repository, detection also inspects the grandparent and rejects the parent as a workspace root if the grandparent exposes multiple git child repositories.

That extra check causes false negatives in practical meta-repo layouts such as:

- `/parent-workspace/ww`
- `/parent-workspace/another-child-repo`
- `/some-grandparent/parent-workspace`
- `/some-grandparent/another-repo`

From `ww`, the intended workspace root is `parent-workspace`, but the grandparent guard forces single-repo mode instead.

Two approaches were considered:

- **A. Strict one-up/one-down anchor**: Always reason from the current repo's main root; if its parent is a git repo, accept that parent immediately as the workspace root. This is simple and matches the original intuition, but it leaves less room for slightly deeper layouts where the containing workspace is one more level up.
- **B. Nearest containing workspace within a bounded window**: Resolve the current repo's main root, then search upward by at most two levels for workspace-root candidates. A candidate qualifies only if it contains the current main repo root and has at least two immediate child real git repositories. Candidates are tested in order: current directory first, then parent of the main repo root, then grandparent of the main repo root. If multiple candidates qualify, pick the nearest one. If the current directory itself already qualifies as a workspace root, accept it immediately. This preserves locality while handling the practical "repo inside repo inside parent directory" cases without recursing arbitrarily.

## Decision

Choose **B**.

Workspace detection is anchored on the current repository's **main worktree root**. Detection may inspect at most:

- the current directory as an immediate workspace-root candidate
- the main repo root
- the parent of the main repo root
- the grandparent of the main repo root

Within that bounded window:

- A workspace-root candidate must contain the current main repo root.
- A workspace-root candidate must expose at least two immediate child **real git repositories**.
- If multiple candidates qualify, the nearest containing candidate wins.
- Managed git worktrees are not treated as real git repositories for workspace detection.

This choice is primarily about matching human expectation in practical 1-2-3 level layouts. When a user is operating from level 3, they may still reasonably expect either level 2 or level 1 to be treated as the active workspace, depending on which directory actually behaves as the containing workspace. Approach A only handles the nearest parent-style case and fails when level 1 is the expected workspace. Approach B preserves that expectation by allowing both containing levels to be considered within a bounded window, then selecting the nearest directory that truly qualifies as a workspace root. That "nearest qualifying workspace wins" rule is also more intuitive than a hard-coded parent-only rule when both level 1 and level 2 could plausibly be treated as workspace containers.

## Consequences

- **Positive**: Restores the intended local mental model while avoiding the current false-negative behavior for git-backed workspace roots.
- **Positive**: Detection remains bounded and deterministic; it does not recurse through arbitrary ancestors.
- **Negative**: The rules are more complex than the simple one-up/one-down model and require explicit tie-breaking.
- **Negative**: Future clone-based isolation (`FR-16`) must mark `ww`-managed clones so they are excluded from "real git repository" detection.
