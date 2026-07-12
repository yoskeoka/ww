# Config type layering: worktree.Config vs flat fields

## Status

Accepted on 2026-03-18.

## Context

`worktree.Manager` depended on `internal/config.Config`, making the public `worktree` package unusable as a library. Three options were considered:

- **Option A**: Move `internal/config` to a public package. Rejected — it couples library consumers to TOML parsing and file search logic they don't need.
- **Option B**: Replace `Config` with flat fields on `Manager`. Simple, but fields will grow as features are added (Phase 2 multi-repo, Phase 4 hooks), bloating the Manager struct.
- **Option C**: Create a public `worktree.Config` type that Manager accepts. CLI layer maps `internal/config.Config` → `worktree.Config` in `newManager()`.

## Decision

Option C. Two separate config types for two separate concerns:

- `worktree.Config` — what Manager needs to operate (public, stable API for library consumers)
- `internal/config.Config` — how `.ww.toml` is parsed (CLI concern, free to change format/fields)

## Consequences

- **Positive**: Library consumers can construct `worktree.Config` without importing internal packages. Config fields stay grouped. Adding new config fields doesn't pollute Manager's top-level API.
- **Negative**: DTO mapping in `newManager()` is boilerplate. Acceptable — it's a handful of field assignments in one place, and it makes the layer boundary explicit.
