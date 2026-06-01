# 0028: Phase 5 global config base

> **Execution**
> Run `/execute-task` for this plan.

## Objective

Set the base rules for Phase 5 global config in `ww`. Cover the user config path, repo-local vs global precedence, full override per key, and sandbox guidance.

## Context

- `docs/project-plan.md` defines Phase 5 as FR-27 through FR-30.
- `docs/specs/configuration.md` and `internal/config/config.go` now load one repo-local `.ww.toml`.
- `post_create_hook` already runs as trusted shell input.
- Human direction for this plan:
  - Global config default path is `$HOME/.config/ww/config.toml`.
  - If `XDG_CONFIG_HOME` is set, use `$XDG_CONFIG_HOME/ww/config.toml` instead.
  - Repo-local config found via existing search rules has higher priority than global config.
  - Matching keys override completely rather than merging deeply or appending.
  - Hook values replace the whole lower layer.

## Options and Trade-offs

1. **Use global config as defaults and repo-local config as full override (recommended)**
   - Pros: Simple mental model. Keeps existing repo-local behavior authoritative. Avoids surprising hook composition and duplicate `copy_files` / `symlink_files` outcomes.
   - Cons: Users must repeat full arrays or hook strings.
2. **Merge some fields and append some arrays**
   - Pros: Less repetition for additive lists.
   - Cons: Harder to explain and test. It also changes hook and file setup behavior in less obvious ways.
3. **Use global config only when no repo-local config exists**
   - Pros: Minimal implementation.
   - Cons: Too weak for the portability goal.

**Recommended option** Put global config on the lower layer. Let repo-local config replace each matching key.
**Decision (2026-06-01)**: Human confirmed XDG-aware global config discovery plus repo-local-wins precedence with complete override semantics for all keys, including arrays and hooks.

## Scope

### In Scope

- Define the global config search path and precedence order
- Specify the layer model between global config and repo-local config
- Define full-override merge semantics for scalar, array, and hook fields
- Document safe sandbox guidance for read-only global config usage
- Introduce the code seams needed for later Phase 5 child plans

### Out of Scope

- Project-target matching rules beyond the foundation hooks needed by Phase 5-02
- Reusable profile syntax beyond the loader/model support needed by Phase 5-03
- Hook trust hardening such as confirmations or dangerous-pattern scanning. That belongs to Phase 7.

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/configuration.md` | Add global config path rules, XDG behavior, precedence, and full-override rules |
| `docs/specs/cli-commands.md` | Note any visible effect of global config loading on create/list/remove |
| `docs/specs/workspace-discovery.md` | Clarify sandbox-boundary behavior if needed |
| `docs/project-plan.md` | Mark FR-27 and FR-30 status updates only if this child plan completes those requirements fully |

## Design Decision Changes

| File | Change |
|------|--------|
| `docs/design-decisions/adr.md` | Add an ADR for the global config path, precedence, and full-override model |

## Code Changes

| File | Change |
|------|--------|
| `internal/config/config.go` | Add global config loading, layering, and precedence |
| `internal/config/config_test.go` | Add unit tests for XDG path selection, `$HOME/.config/ww`, and override rules |
| `cmd/ww/main.go` | Pass the layered config result into manager setup |
| `worktree/worktree.go` | Accept any needed config model additions |

## Test Changes

- Add loader tests covering:
  - `XDG_CONFIG_HOME` set
  - `XDG_CONFIG_HOME` unset with `$HOME/.config/ww/config.toml`
  - no global config present
  - repo-local config overriding scalar fields
  - repo-local config overriding array fields without append
  - repo-local config overriding hook fields without composition
- Add command-level regression coverage proving current repo-local-only behavior remains unchanged when no global config file exists.

## Design Notes

- The precedence rule should stay explicit and uniform. The higher-priority config replaces the lower-priority value for the same key.
- Skip implicit append behavior in Phase 5.
- Global config should remain optional. Existing repositories that rely only on `.ww.toml` must continue to work unchanged.
- Sandbox guidance should prefer readable global config and avoid recommending writable global config paths to agent sandboxes.

## Sub-tasks

- [ ] [parallel] Update `docs/specs/configuration.md` with global path, XDG behavior, precedence order, and complete-override semantics
- [ ] [parallel] Append ADR for global config layering and trust boundary
- [ ] [depends on: specs, ADR] Add global config discovery and deterministic layering in `internal/config`
- [ ] [depends on: implementation] Thread the layered config result through manager setup without breaking current behavior
- [ ] [depends on: implementation] Add loader and command regression tests for XDG/global/repo-local precedence
- [ ] [depends on: verification] Decide whether FR-27 and FR-30 are done now or stay partial until later Phase 5 child plans land

## Verification

- `go test ./internal/config ./cmd/ww ./worktree`
- Manual verification:
  - with only global config present, `ww` picks it up
  - with both global and repo-local config present, repo-local values win per key
  - array and hook fields replace the lower layer as a whole
  - when `XDG_CONFIG_HOME` is set, `ww` prefers `$XDG_CONFIG_HOME/ww/config.toml`
  - when `XDG_CONFIG_HOME` is unset, `ww` falls back to `$HOME/.config/ww/config.toml`
