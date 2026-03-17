# 006: Decouple worktree.Manager from internal/config

**Objective**: Remove `worktree.Manager`'s dependency on `internal/config.Config` so that the `worktree` package is usable as a library by external consumers. Closes [GH #5](https://github.com/yoskeoka/ww/issues/5).

**Approach**: Option B from the issue — replace `Config *config.Config` with plain fields on `Manager`. The CLI layer (`cmd/ww/`) maps config values to Manager fields. `internal/config` stays internal (it handles TOML parsing, file search — CLI concerns).

## Spec changes

- [ ] `docs/specs/configuration.md` — no change needed (describes file format, not Go types)
- [ ] `docs/specs/cli-commands.md` — no change needed (describes user-facing behavior)

No spec changes required. This is a pure internal refactor — public CLI behavior is unchanged.

## Sub-tasks

- [ ] [parallel] Add plain fields to `worktree.Manager` replacing `Config *config.Config`:
  - `WorktreeDir string`
  - `DefaultBase string`
  - `CopyFiles []string`
  - `SymlinkFiles []string`
  - `PostCreateHook string`
- [ ] [parallel] Update all `m.Config.X` references in `worktree/worktree.go` to `m.X`
- [ ] [depends on: above] Update `cmd/ww/main.go:newManager()` to map `config.Config` fields to `Manager` fields
- [ ] [depends on: above] Remove `config` import from `worktree` package
- [ ] [depends on: above] Run `make test` — all existing tests must pass with zero changes
- [ ] Move issue file to `docs/issues/done/`

## Design decisions

No ADR needed. This is a straightforward decoupling — Option B was already identified in the issue as the cleaner approach for library consumers.
