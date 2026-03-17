# 006: Decouple worktree.Manager from internal/config

**Objective**: Remove `worktree.Manager`'s dependency on `internal/config.Config` so that the `worktree` package is usable as a library by external consumers. Resolves `docs/issues/public-worktree-depends-on-internal-config.md`.

**Approach**: Option C — create a public `worktree.Config` type in the `worktree` package that `Manager` accepts. `internal/config` stays internal for TOML parsing and file search. The CLI layer (`cmd/ww/newManager()`) maps `internal/config.Config` to `worktree.Config` (DTO transfer).

This keeps config fields grouped (not scattered as flat Manager fields), and cleanly separates:
- `worktree.Config` — what the Manager needs to operate (public, library-friendly)
- `internal/config.Config` — how `.ww.toml` is parsed (CLI concern, internal)

## Spec changes

None required. This is a pure internal refactor — public CLI behavior is unchanged.

## Sub-tasks

- [ ] [parallel] Define `worktree.Config` struct in `worktree/worktree.go` with fields: `WorktreeDir`, `DefaultBase`, `CopyFiles`, `SymlinkFiles`, `PostCreateHook`
- [ ] [parallel] Replace `Manager.Config *config.Config` with `Manager.Config Config` (the new `worktree.Config`)
- [ ] [depends on: above] Update all `m.Config.X` references (field names stay the same, just the type changes)
- [ ] [depends on: above] Remove `internal/config` import from `worktree` package
- [ ] [depends on: above] Update `cmd/ww/main.go:newManager()` to map `internal/config.Config` → `worktree.Config`
- [ ] [depends on: above] Run `make test` — all existing tests must pass
- [ ] Move `docs/issues/public-worktree-depends-on-internal-config.md` to `docs/issues/done/`

## Code Changes

| File | Changes |
|------|---------|
| `worktree/worktree.go` | Add `Config` struct; change `Manager.Config` type from `*config.Config` to `Config`; remove `internal/config` import |
| `cmd/ww/main.go` | In `newManager()`, map `internal/config.Config` fields to `worktree.Config` fields |

## Design decisions

Option C over Option B (flat fields on Manager): Config fields will grow as features are added. Grouping them in `worktree.Config` keeps Manager clean and makes the boundary between "what Manager needs" and "how config is loaded" explicit.
