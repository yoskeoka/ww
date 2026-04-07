# 015: Phase 3 — User-Facing Documentation

> **Execution**: Use `/execute-task` to implement this plan.

**Parent plan**: `docs/exec-plan/todo/phase3-polish.md`

**Objective**: Write first-release quality user documentation covering installation, shell integration setup, common workflows, and workspace-mode behavior. This plan depends on 013 (shell integration) and 014 (install/versioning) being completed first, as docs must reflect actual implemented behavior.

## Dependencies

- **013-phase3-shell-integration**: `ww cd` and `ww create -q` must be implemented before documenting them
- **014-phase3-install-and-versioning**: `go install`, `brew install`, and version output must be finalized before writing install instructions

## Spec Changes

None. This plan documents existing and newly-implemented behavior; it does not introduce new CLI features.

## Code Changes

None. Documentation only.

## Docs Changes

| File | Change |
|------|--------|
| `README.md` | Rewrite for end-users: install instructions (`go install`, Homebrew tap), quick start, shell integration examples, workspace mode overview |
| `docs/specs/shell-integration.md` | Add shell wrapper examples section if not already covered by 013 |
| `docs/spec-code-mapping.md` | Update mappings for any new specs added by 013/014 |
| `docs/specs/README.md` | Link new specs (shell-integration, release-versioning) |

### README Structure

1. **Overview** — one-paragraph description of what `ww` does
2. **Install**
   - `brew tap yoskeoka/ww && brew install ww`
   - `go install github.com/yoskeoka/ww/cmd/ww@latest`
3. **Quick Start** — single-repo usage: `ww create`, `ww list`, `ww remove`
4. **Shell Integration**
   - `ww cd` usage and shell wrapper setup (`wcd` function for bash/zsh)
   - `ww create -q` with command substitution: `cd $(ww create -q feat/x)`
5. **Workspace Mode** — multi-repo setup with `.ww.toml`, `ww list` across repos, `ww clean`
6. **Configuration** — `.ww.toml` reference (hooks, file copying)
7. **Commands** — brief reference table linking to full spec

### Writing Guidelines

- Write for a developer audience who knows git but has never used worktrees
- Include copy-paste-ready shell snippets
- Keep it concise — link to specs for exhaustive details rather than duplicating them
- Verify every code example actually works against the built binary before finalizing

## Sub-tasks

- [ ] [parallel] Draft README.md structure and install section
- [ ] [parallel] Draft shell integration section with bash/zsh wrapper examples
- [ ] [depends on: drafts] Write workspace mode section with `.ww.toml` example
- [ ] [depends on: drafts] Write commands reference table
- [ ] [depends on: all sections] Review all code examples against built binary
- [ ] [depends on: review] Update `docs/specs/README.md` and `docs/spec-code-mapping.md`

## Verification

- All install commands in README work on a clean environment
- All shell snippets in README are copy-paste-ready and produce expected output
- `ww cd` and `ww create -q` examples match actual CLI behavior
- Workspace mode example matches current `.ww.toml` config format
- No broken links in docs
- `make lint` passes (no code changes, but verify nothing was accidentally modified)
