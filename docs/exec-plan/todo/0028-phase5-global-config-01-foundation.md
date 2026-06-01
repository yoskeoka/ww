# 0028: Phase 5 Global Config Foundation

> **Execution**: Use `/execute-task` to implement this plan.

## Objective

Establish the Phase 5 global-config foundation for `ww`: a user-owned config location, deterministic precedence between repo-local and global config, full-override merge semantics, and sandbox-safe guidance for loading global settings without requiring every repository to commit `.ww.toml`.

## Context

- `docs/project-plan.md` defines Phase 5 as global config and hook workflow portability, specifically FR-27 through FR-30.
- `docs/specs/configuration.md` and `internal/config/config.go` currently model configuration as a single repo-local `.ww.toml` discovered from the current directory upward, with a few explicit fallback directories.
- Existing config is intentionally simple and trusted, and `post_create_hook` already executes as trusted shell input. Phase 5 should extend that model cautiously rather than introducing implicit config composition that is hard to reason about.
- Human direction for this plan:
  - Global config default path is `$HOME/.config/ww/config.toml`.
  - If `XDG_CONFIG_HOME` is set, use `$XDG_CONFIG_HOME/ww/config.toml` instead.
  - Repo-local config found via existing search rules has higher priority than global config.
  - Matching keys override completely rather than merging deeply or appending.
  - Hook values must be replaced as a whole, not partially composed.

## Options and Trade-offs

1. **Global config as defaults, repo-local config as full override (recommended)**
   - Pros: Simple mental model. Keeps existing repo-local behavior authoritative. Avoids surprising hook composition and duplicate `copy_files` / `symlink_files` outcomes.
   - Cons: Users must repeat full arrays or hook strings when overriding only one element.
2. **Per-field merge with array append for selected keys**
   - Pros: Less repetition for additive lists.
   - Cons: Harder to explain and test. Implicitly changes hook/materialization behavior when global and local config interact. Duplicate entries, ordering, and partial override rules become policy decisions instead of straightforward precedence.
3. **Global config only when no repo-local config exists**
   - Pros: Minimal implementation.
   - Cons: Too weak for the stated portability goal because repo-local config cannot override a shared user baseline selectively.

**Recommendation**: Option 1. Treat global config as a lower-priority default layer and repo-local config as a full replacement per key. Defer any additive semantics to a future explicit opt-in design if real usage proves the need.
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
- Hook trust hardening such as confirmations or dangerous-pattern scanning; that belongs to Phase 7

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/configuration.md` | Add global config path rules, XDG behavior, precedence order, and full-override merge semantics |
| `docs/specs/cli-commands.md` | Document any CLI-visible consequences of global config loading, especially when config source affects create/list/remove behavior |
| `docs/specs/workspace-discovery.md` | Clarify any sandbox-boundary interaction with global config lookup if the execution design needs it |
| `docs/project-plan.md` | Mark FR-27 and FR-30 status updates only if this child plan completes those requirements fully |

## Design Decision Changes

| File | Change |
|------|--------|
| `docs/design-decisions/adr.md` | Append ADR for the global config location, precedence order, and complete-override layering model |

## Code Changes

| File | Change |
|------|--------|
| `internal/config/config.go` | Extend config loading to support a global config source, layering, and source-aware precedence |
| `internal/config/config_test.go` | Add unit coverage for XDG path selection, fallback to `$HOME/.config/ww`, and override semantics |
| `cmd/ww/main.go` | Thread the layered config result into manager construction without changing existing command contracts |
| `worktree/worktree.go` | Accept any required config model additions while preserving current behavior for existing repo-local configs |

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

- The precedence rule should be explicit and uniform: the higher-priority config replaces the lower-priority value for the same key, regardless of value type.
- Do not introduce implicit append behavior in Phase 5. If additive reuse becomes necessary later, design it as explicit syntax instead of hidden merge magic.
- Global config should remain optional. Existing repositories that rely only on `.ww.toml` must continue to work unchanged.
- Sandbox guidance should prefer readable global config and avoid recommending writable global config paths to agent sandboxes.

## Sub-tasks

- [ ] [parallel] Update `docs/specs/configuration.md` with global path, XDG behavior, precedence order, and complete-override semantics
- [ ] [parallel] Append ADR for global config layering and trust boundary
- [ ] [depends on: specs, ADR] Implement global config discovery and deterministic layering in `internal/config`
- [ ] [depends on: implementation] Thread the layered config result through manager setup without breaking current behavior
- [ ] [depends on: implementation] Add loader and command regression tests for XDG/global/repo-local precedence
- [ ] [depends on: verification] Decide whether FR-27 and FR-30 can be marked implemented immediately or remain partial until later Phase 5 child plans land

## Verification

- `go test ./internal/config ./cmd/ww ./worktree`
- Manual verification:
  - with only global config present, `ww` picks it up
  - with both global and repo-local config present, repo-local values win per key
  - array and hook fields are replaced completely, not appended or composed
  - when `XDG_CONFIG_HOME` is set, `ww` prefers `$XDG_CONFIG_HOME/ww/config.toml`
  - when `XDG_CONFIG_HOME` is unset, `ww` falls back to `$HOME/.config/ww/config.toml`
