# Architectural Decision Records (ADR)

## [YYYY-MM-DD] Title of Decision

### Context
[Describe the issue or problem.]

### Decision
[Describe the decision made.]

### Consequences
[Describe the positive and negative consequences.]

---

## [2026-03-18] Config type layering: worktree.Config vs flat fields

### Context

`worktree.Manager` depended on `internal/config.Config`, making the public `worktree` package unusable as a library. Three options were considered:

- **Option A**: Move `internal/config` to a public package. Rejected — it couples library consumers to TOML parsing and file search logic they don't need.
- **Option B**: Replace `Config` with flat fields on `Manager`. Simple, but fields will grow as features are added (Phase 2 multi-repo, Phase 4 hooks), bloating the Manager struct.
- **Option C**: Create a public `worktree.Config` type that Manager accepts. CLI layer maps `internal/config.Config` → `worktree.Config` in `newManager()`.

### Decision

Option C. Two separate config types for two separate concerns:

- `worktree.Config` — what Manager needs to operate (public, stable API for library consumers)
- `internal/config.Config` — how `.ww.toml` is parsed (CLI concern, free to change format/fields)

### Consequences

- **Positive**: Library consumers can construct `worktree.Config` without importing internal packages. Config fields stay grouped. Adding new config fields doesn't pollute Manager's top-level API.
- **Negative**: DTO mapping in `newManager()` is boilerplate. Acceptable — it's a handful of field assignments in one place, and it makes the layer boundary explicit.

---

## [2026-03-18] Workspace detection algorithm: parent-scan strategy

### Context

Phase 2 adds workspace awareness — detecting when `ww` is inside a meta-repo with multiple child git repositories. Several detection strategies were considered:

- **Config-only**: Require explicit `workspace = true` in `.ww.toml`. Simple but violates "convention over configuration" — new users must configure before workspace features work.
- **Recursive scan**: Walk up the directory tree and scan all children at each level. Correct but slow and risks false positives on deeply nested repos.
- **Parent-scan (chosen)**: Check only the immediate parent directory for git siblings and the current directory for git children. Limited to one level in each direction.

### Decision

Parent-scan strategy with a 7-step algorithm:
1. Scan CWD children for `.git` entries (parent candidate)
2. Determine current git repo root
3. Check parent directory for `.git`
4. Check parent's children for `.git` siblings
5. (Reserved for future config override)
6. Fall back to CWD as workspace root if step 0 found children
7. None → single-repo mode

### Consequences

- **Positive**: Fast (only two directory scans), predictable, zero-config for standard meta-repo layouts.
- **Negative**: Cannot detect workspaces more than one level deep. Accepted — FR-22 (recursive detection) is reserved for the future if needed.

---

## [2026-03-18] Worktree path layout: centralized `.worktrees/` in workspace mode

### Context

In single-repo mode, worktrees are created as siblings: `<repo-parent>/<repo>@<branch>`. In workspace mode with N child repos, sibling layout would scatter worktrees across child directories, making them hard to find and clean up.

### Decision

Workspace mode centralizes all worktrees under `<workspace_root>/.worktrees/<repo>@<branch>`. Single-repo mode keeps the sibling layout. An explicit `worktree_dir` in `.ww.toml` overrides either default.

### Consequences

- **Positive**: Single location for all worktrees across all repos. Easier cleanup, listing, and mental model.
- **Negative**: Worktrees are not adjacent to their source repo. Accepted — `ww list` provides the lookup and most editors/agents use absolute paths.

---

## [2026-03-18] Child repos are never workspace roots (no recursive nesting)

### Context

If a child repo could itself be treated as a workspace root (e.g., a child that has its own git children), workspace detection could recurse indefinitely or produce ambiguous results.

### Decision

Child repos are never treated as workspace roots. Detection stops at one level. This is an explicit invariant enforced in the detection algorithm.

### Consequences

- **Positive**: Deterministic detection, no ambiguity, simple mental model.
- **Negative**: Nested workspace-in-workspace layouts are not supported. Accepted — FR-22 reserves this for the future.

---

## [2026-03-18] Worktree STATUS: merged > stale, no-tracking = active

### Context

`ww list` adds a STATUS column (`active`/`merged`/`stale`). A branch could theoretically be both merged and stale (merged into base AND remote branch deleted). Branches without remote tracking could be considered stale by absence, but this would flag local-only feature branches incorrectly.

### Decision

- `merged` takes precedence over `stale` (if both conditions are met).
- Branches with no remote tracking configured are always `active`, never `stale`.
- Main worktrees are always `active`.

### Consequences

- **Positive**: Conservative — only flags worktrees as cleanable when there's strong evidence. Avoids false positives on local-only branches.
- **Negative**: A branch that was never pushed will stay `active` even if abandoned. Accepted — FR-23 (time-based stale detection) can address this later.

---

## [2026-03-18] `ww clean` has no confirmation prompt

### Context

Most CLI tools that delete data ask for confirmation (`rm -i`, `git clean -i`). However, `ww clean` targets a specific audience (developers and AI agents) and has explicit preview mechanisms.

### Decision

`ww clean` executes immediately without a confirmation prompt. Users preview with `ww list --cleanable` or `ww clean --dry-run`.

### Consequences

- **Positive**: Scriptable, agent-friendly, no interactive input required. Consistent with `ww remove` which also has no prompt.
- **Negative**: Risk of accidental deletion. Mitigated by safe defaults (`git branch -d` fails on unmerged branches) and `--force` being opt-in.
