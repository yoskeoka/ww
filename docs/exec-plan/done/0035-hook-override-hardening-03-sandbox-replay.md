# 0035: Hook Override Hardening 03 Sandbox Replay
> **Execution**: Use `/execute-task` to implement this plan. After implementation is complete, use `/review-task` to prepare and create the PR.

**Shared branch**: `plan/hook-override-hardening`

## Objective

Add an explicit post-create replay surface so a human can re-run worktree materialization after sandbox-restricted startup blocked `copy_files`, `symlink_files`, or create-time hooks. Completion means a user can enter the created worktree and invoke a documented `ww` command to preview and then execute the selected repo's effective post-create materialization again, with optional `ww i` visibility over the same actions, instead of losing setup permanently when sandbox reads were denied during the original `ww create`.

## Context

- In sandboxed agent sessions, create-time materialization can fail on files such as `.env` or `.git/config` even though the worktree itself was created successfully.
- The requested recovery flow is explicit, not automatic: no dir-change hooks, no background retry, and no hidden state mutation after `ww create` exits.
- This plan should reuse the effective config and lifecycle work from `0033` and `0034` instead of inventing a second hook/materialization engine.

## Existing Implementation References

- `worktree/worktree.go`
  - create flow materialization order, lines 224-229
  - remove flow boundaries, lines 586-641
  - copy/symlink helpers, lines 658-688
  - `runPostCreateHook`, lines 690-713
- `cmd/ww/sub_interactive.go`
  - interactive flow wiring, lines 60-124
  - create preview and execution helpers, lines 223-265
- `internal/interactive/create_flow.go`
  - create preview contract, lines 9-31
- `internal/interactive/huh_ui.go`
  - create preview rendering, lines 306-326
- `docs/specs/cli-commands.md`
  - command parity rule and current `ww i` create contract, lines 180-260
- `docs/specs/interactive-mode.md`
  - interactive non-negotiable invariant, lines 16-21
  - create flow contract, lines 97-130
- `docs/specs/configuration.md`
  - sandbox config search guidance, lines 124-140

## Code Change Map

- `cmd/ww/sub_hook.go` (NEW)
  - add an explicit replay-oriented subcommand for post-create materialization from inside a worktree or via an explicit branch/repo target
- `cmd/ww/main.go` (MODIFY)
  - register the new command and ensure it resolves repo/worktree context with the same sandbox-aware rules as the rest of the CLI
- `worktree/worktree.go` (MODIFY)
  - extract reusable post-create materialization/replay logic that can preview and execute copy, symlink, and post-create hook steps without re-running `git worktree add`
- `cmd/ww/sub_interactive.go` (MODIFY)
  - optionally expose the replay flow in `ww i` using the same non-interactive command semantics and preview data
- `internal/interactive/create_flow.go` (MODIFY)
  - extend preview/execution structures only as needed for explicit replay parity
- `internal/interactive/huh_ui.go` (MODIFY)
  - render replay-targeted copy/symlink/hook visibility for human confirmation
- `cmd/ww/sub_hook_test.go` (NEW)
  - add command-level coverage for preview, execution, and targeting behavior
- `worktree/worktree_test.go` (MODIFY)
  - add replay-materialization coverage without re-creating the worktree

## Spec Changes

- `docs/specs/cli-commands.md`
  - document the new explicit replay command, its preview/execution behavior, and its sandbox recovery intent
- `docs/specs/interactive-mode.md`
  - document any new `ww i` replay entry point or replay preview contract while preserving the non-interactive parity invariant

## Sub-tasks

- [x] Define the explicit replay command surface and argument model for post-create recovery
- [x] Extract reusable post-create materialization logic that can:
  - preview copy/symlink/hook steps
  - execute them against an existing worktree
  - reuse the target repo's effective config from `0033`
  - reuse lifecycle hook execution from `0034`
- [x] Add command-level tests covering:
  - running from inside the current worktree
  - explicit repo/branch targeting when supported
  - preview-only output for copy, symlink, and hook steps
- [x] If interactive support is included, surface the same replay actions and confirmation data through `ww i` without introducing unique behavior (not included; standard command parity is documented)
- [x] Update CLI and interactive specs so sandbox recovery is documented as an explicit human-triggered flow

## Design Decisions

- Prefer explicit replay over automatic retries. Users should decide when elevated permissions or broader filesystem access are acceptable.
- Re-evaluate the current effective config at replay time rather than introducing a persisted per-worktree materialization manifest in this phase. That keeps the feature stateless and aligned with the existing config-driven model.
- Limit the replay surface to post-create materialization. Pre-create and pre-remove hooks are lifecycle-bound and should not be re-run after the fact by this recovery command.

## Parallelism

- Depends on `0033-hook-override-hardening-01-child-repo-overrides.md`.
- Depends on `0034-hook-override-hardening-02-lifecycle-context.md`.
- After those two land, CLI command work and optional interactive parity can proceed in parallel if the shared replay helper API is settled first.
