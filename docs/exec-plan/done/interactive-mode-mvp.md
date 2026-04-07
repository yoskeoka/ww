# Interactive Mode MVP (`ww i`)

> **Execution**: Use `/execute-task` on the child plans listed in "Plan Split". Do not implement directly from this parent plan.

**Objective:** Define the Phase 4 interactive-mode MVP as an umbrella plan, including the core invariants, CLI parity rule, and implementation split needed to execute `ww i` safely without inventing behavior that only exists in the interactive UI.

## Context

Phase 2 established `ww`'s main strength: workspace-wide visibility across repositories. The interactive mode must preserve that advantage instead of forcing users into repo-first navigation.

The desired MVP is:

- command name: `ww i`
- lightweight prompt flow, not a full-screen TUI
- initial screen shows detected mode and repo context, then asks which operation to run
- menu scope: `create`, `list`, `clean`, `quit`
- `list` operates on **worktrees**, not repos, with interactive filtering
- `list` actions: `open`, `remove`, `back`
- `open` writes the selected path to `stdout` only
- `remove` is available only from `list`; there is no top-level `remove` menu
- `clean` shows a repo-level summary first, then a detailed confirmation view
- batch or multi-select operations are explicitly out of scope

## Absolute Principle: Interactive Execution Parity

Interactive mode is an alternate input surface for humans, not a separate capability surface.

- Every interactive action that causes an effect outside the prompt itself MUST correspond to an equivalent non-interactive `ww` command plus option combination that can produce the same result.
- "Equivalent" means the user can achieve the same filesystem/git outcome without using `ww i`.
- Preview-only UI steps do not need a 1:1 command, but any action that opens, creates, removes, or cleans MUST have CLI parity.
- If implementation of `ww i` reveals an action that cannot be expressed by the existing non-interactive CLI, the required command/flag combination MUST be specified and implemented as part of the same overall effort before the interactive action is considered complete.
- Interactive mode may compose existing commands internally, but it must not become the only place where a capability exists.

This rule preserves `ww`'s AI/scripting contract and prevents UX-only behavior from bypassing the normal command surface.

### Current Parity Mapping

| Interactive action | Required non-interactive equivalent | Notes |
|--------------------|-------------------------------------|-------|
| `create` | `ww create <branch>` and `ww create --repo <repo> <branch>` | Existing parity path |
| `list` browsing | `ww list` / `ww list --cleanable` | Interactive filtering is a UI affordance over existing list data |
| `open` selected worktree | `ww cd <branch>` and `ww cd --repo <repo> <branch>` | `stdout` path-only contract already exists |
| `remove` from list | `ww remove <branch>` and `ww remove --repo <repo> <branch>` | Main worktree cannot be removed, matching existing CLI rule |
| `clean` confirmed execution | `ww clean` and `ww clean --force` | Interactive summary/confirmation is additive UX over existing mutation |

### Gap Handling Rule

Before any new interactive action is added beyond the table above, the plan/spec must first answer:

1. What exact non-interactive command expresses the same operation?
2. Does that command already exist?
3. If not, which child plan adds it first?

If those answers are missing, the interactive action is out of scope for the MVP.

## Reviewed Decisions / Constraints

Past decisions reviewed before planning:

- `docs/design-decisions/core-beliefs.md`: AI-first remains intact because `ww i` is additive; non-interactive commands remain the primary agent/scripting interface.
- `docs/design-decisions/adr.md`: existing git-native and workspace-aware decisions still apply. The interactive mode must call the same underlying logic as standard subcommands rather than inventing separate behavior.

New design choices captured by this plan:

- use `ww i` rather than `ww ui` / `ww interactive`
- use a lightweight prompt library rather than Bubble Tea for the MVP
- prefer `huh` over `promptui` / `survey`
  - `survey` is out because its official README states it is no longer maintained
  - `promptui` is viable for one-off prompts, but this feature is a multi-step guided flow
- `huh` fits grouped step flows better and already includes filterable selection behavior suitable for the first finder implementation
- do not implement a custom fuzzy-ranking engine in Phase 4 MVP; use filterable selection and revisit only if the experience is insufficient

Past decision: `docs/design-decisions/core-beliefs.md` says AI-first takes precedence, and `docs/design-decisions/adr.md` already established explicit path-only shell interfaces (`ww cd`, `ww create -q`) rather than hidden shell-state mutation. The same reasoning applies here: interactive mode must stay additive and composable, not become a parallel control plane with unique powers.

## Scope

### In Scope

- New `ww i` subcommand
- TTY-only execution
- Mode/repo overview before operation selection
- Guided `create` flow
- Guided `list` flow with filterable worktree selection
- Guided `clean` flow with preview/confirmation
- Path-only `stdout` result for `open`
- Explicit handling of main worktrees in list results

### Out of Scope

- Auto-entering interactive mode when `ww` is run without subcommands
- Full-screen TUI / Bubble Tea app
- Batch or multi-select actions
- A top-level `remove` menu
- Custom fuzzy-scoring or fzf parity
- JSON mode support inside `ww i`; for the MVP, invoking `ww i --json` MUST fail fast with a clear error explaining that interactive mode does not support `--json` and that users should use non-interactive commands for JSON output

## UX Specification

### Entry / Preconditions

- `ww i` requires an interactive terminal for its interactive input and prompts. Standard input and the primary prompt/render stream MUST be TTYs.
- `ww i` MAY be invoked with `stdout` redirected or non-TTY. When `stdout` is non-TTY, all interactive prompts, context, menus, previews, confirmations, and human-readable errors MUST be written to `stderr`, and `stdout` MUST remain path-only for the `open` action so it is safe to use in command substitution / piping.
- If no TTY is available for interactive input/prompts (for example, stdin is not a TTY and there is no TTY output stream), fail immediately with a clear error that tells the user to use standard commands and consult `ww --help`.
- Workspace detection uses the existing Phase 2 logic. No separate workspace model is introduced.

### Initial Screen

Before asking for the action, show lightweight context:

- detected mode: `workspace` or `single-repo`
- current workspace root or repo root
- repo list when in workspace mode

Then prompt for one action:

- `create`
- `list`
- `clean`
- `quit`

### Create Flow

1. Resolve target repo:
   - workspace mode: choose repo interactively
   - single-repo mode: use the current repo without prompting
2. Enter branch information
3. Show preview:
   - target worktree path
   - base branch
   - copy/symlink actions
   - hook execution when configured
4. Confirm
5. Execute using the existing create logic

CLI parity requirement:

- The confirmed action MUST be equivalent to `ww create [--repo <repo>] <branch>`.
- If interactive UX later adds base selection, branch reuse policy, or any other create-time decision that is not currently expressible through `ww create`, that CLI surface must be added first. Such expansion is out of scope for this MVP parent plan.

### List Flow

1. Build the workspace-wide worktree list using existing list/status logic
2. Present a filterable selector over **worktrees**
3. Display fields:
   - repo
   - branch
   - status
   - shortened path
   - main-worktree marker when applicable
4. Search/filter against:
   - repo
   - branch
   - status
   - full path
5. After selecting a worktree, offer:
   - `open`
   - `remove`
   - `back`
6. Main worktree behavior:
   - `open` allowed
   - `remove` disabled / not offered
   - UI clearly marks it as the main worktree
7. `open` behavior:
   - print the selected path to `stdout` only
   - write all prompts, menus, context, confirmations, and human-readable guidance to `stderr`
8. `remove` behavior:
   - show preview before deletion
   - require confirmation
   - reuse the existing remove logic

CLI parity requirement:

- `open` MUST be equivalent to `ww cd [--repo <repo>] <branch>`.
- `remove` MUST be equivalent to `ww remove [--repo <repo>] <branch>`.
- Since `ww remove` rejects main worktrees, interactive mode must not offer `remove` for main worktrees.
- If worktree selection ever needs to target something that cannot be expressed by branch name plus optional `--repo`, a non-interactive selector command/flag must be introduced first.

### Clean Flow

1. Compute cleanable worktrees using existing merged/stale logic
2. Show a repo-level summary first, including zero-count repos when in workspace mode
3. Ask whether to proceed, and whether removal is safe or forced
4. Show a detailed list of targeted worktrees before final confirmation
5. Execute using the existing clean logic

The clean flow is intentionally more informative than plain `ww clean`; the interactive mode's value is visibility and confirmation.

CLI parity requirement:

- Final execution MUST be equivalent to `ww clean` or `ww clean --force`.
- The interactive summary/detail screens are UX-only and do not require a 1:1 non-interactive text view, as long as the final mutation matches the existing command result.

## Spec Changes

| File | Change |
|------|--------|
| `docs/specs/cli-commands.md` | Add `ww i` command behavior, TTY requirement, and high-level flows |
| `docs/specs/interactive-mode.md` | New spec describing interactive-mode UX, list selector behavior, clean summary/detail flow, and `open` output contract |

## Design Decision Changes

| File | Change |
|------|--------|
| `docs/design-decisions/adr.md` | Record the Phase 4 MVP interaction model: `ww i`, `huh`, filterable worktree selector, no batch/multi-select |

## Code Changes

| File | Change |
|------|--------|
| `cmd/ww/main.go` | Register the new `i` subcommand |
| `cmd/ww/sub_interactive.go` | New interactive-mode entry point and high-level flow wiring |
| `cmd/ww/helpers.go` | Reuse or extend shared repo/workspace resolution helpers as needed |
| `internal/interactive/` | New package for prompt flow, selection formatting, and TTY checks |
| `integration_test.go` | Add interactive-mode integration coverage where feasible |

Exact file layout may shift during implementation, but the interactive flow should live outside `worktree/` so the core business logic remains reusable.

## Plan Split

This parent plan is intentionally not implementation-ready by itself. Execute the work via the following child plans:

| Child plan | Objective | Depends on |
|------------|-----------|------------|
| `docs/exec-plan/todo/020-interactive-mode-contract-and-foundation.md` | Lock spec/ADR/contracts, TTY behavior, stdout/stderr routing, and the shared interaction foundation | none |
| `docs/exec-plan/todo/021-interactive-mode-list-open-remove.md` | Implement the worktree selector flow plus `open`/`remove` actions with CLI parity | 020 |
| `docs/exec-plan/todo/022-interactive-mode-create-clean.md` | Implement guided `create` and `clean` flows with CLI parity | 020 |

Suggested execution order:

1. Land 020 first to freeze the contract.
2. Then 021 and 022 may proceed independently in parallel.
3. If either child plan uncovers missing non-interactive CLI parity, add and land that CLI capability in the relevant child plan before completing the interactive flow.

## Testing Strategy

Interactive flows should not depend solely on manual testing.

- Extract prompt/session logic behind small interfaces so the step transitions are unit-testable without a real terminal.
- Keep git/worktree behavior delegated to existing tested code paths.
- Add at least one non-TTY integration test to verify `ww i` fails with the intended message.
- Add focused tests for:
  - workspace initial screen inputs
  - worktree selector item formatting / filtering data
  - main worktree action restrictions
  - `open` returning path-only `stdout`
  - clean summary and detailed confirmation construction

If full PTY integration proves too expensive in the first pass, prioritize unit-level flow coverage plus non-TTY command verification.

## Sub-tasks

- [ ] [parallel] Refine `docs/specs/cli-commands.md` and add `docs/specs/interactive-mode.md` with the parity rule and `ww i` MVP contract
- [ ] [parallel] Append ADR entry documenting `ww i`, `huh`, the no-unique-capabilities rule, and MVP scope boundaries
- [ ] [depends on: specs, ADR] Create the shared interactive foundation (`ww i` command, TTY gate, stderr/stdout routing, overview screen)
- [ ] [depends on: foundation] Implement list/open/remove flow in a child plan
- [ ] [depends on: foundation] Implement create/clean flow in a child plan
- [ ] [depends on: list/open/remove, create/clean] Add unit and integration coverage for shared flow helpers and command behavior

## Verification

- No interactive mutation exists without an equivalent non-interactive `ww` command + option combination
- `ww i` starts only in interactive terminals
- Workspace mode preserves repo-wide visibility before action selection
- `list` filters worktrees, not repos
- Main worktrees are selectable for `open` and unavailable for `remove`
- `open` writes only the selected path to `stdout`
- `clean` shows summary + detailed confirmation before execution
- Existing non-interactive commands remain unchanged
