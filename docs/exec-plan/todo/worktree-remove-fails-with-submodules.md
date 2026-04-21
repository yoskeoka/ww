# Worktree Remove Fails With Submodules
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Make `ww remove` and `ww clean` handle Git's "working trees containing
submodules cannot be moved or removed" failure with a clear, actionable `ww`
diagnostic instead of surfacing a raw `git worktree remove` error.

This resolves `docs/issues/worktree-remove-fails-with-submodules.md` without
implementing a deletion fallback in the planning PR.

## Context

- `docs/project-plan.md` emphasizes Git-native behavior, agent-friendly output,
  and safe mutation previews.
- `docs/design-decisions/core-beliefs.md` prioritizes correctness over speed and
  discourages unrelated refactors.
- `docs/design-decisions/adr.md` already records that `ww clean` has no
  confirmation prompt because users have explicit preview mechanisms, but safe
  defaults still matter.
- The current removal path is shared: `cmd/ww/sub_remove.go` and
  `cmd/ww/sub_clean.go` both call `worktree.Manager.Remove`, which delegates to
  `git.Runner.WorktreeRemove`.

## Reproduction

Use a disposable repository so the cleanup commands are safe to run:

1. Create a parent repo and a local submodule source repo.
2. In the parent repo, commit a file on `main`.
3. Add the local source repo as a submodule, commit `.gitmodules` and the
   submodule entry, and ensure `protocol.file.allow=always` is set for the test
   command if Git rejects local submodule URLs.
4. Run `ww create feat/submodule-remove`.
5. Run `git -C <created-worktree> submodule update --init --recursive` if the
   submodule checkout is not already populated.
6. Run `ww remove --force feat/submodule-remove`.
7. Observe that Git exits 128 with:

   ```text
   fatal: working trees containing submodules cannot be moved or removed
   ```

8. Confirm `ww` currently wraps that as a raw low-level failure similar to:

   ```text
   removing worktree: git worktree remove --force /path/to/worktree: exit status 128
   ```

The same underlying failure can appear through `ww clean` because clean invokes
the same `Manager.Remove` method for each cleanable worktree.

## Trade-off Decision

Choose **guided remediation** for the implementation plan.

| Option | Behavior | Pros | Cons |
| --- | --- | --- | --- |
| Safer descriptive error | Replace the raw Git failure with a short explanation that submodule worktrees cannot be removed by `git worktree remove`. | Lowest risk; avoids accidental deletion. | User still has to know or find the cleanup commands. |
| Guided remediation | Detect the submodule-specific Git failure, explain the limitation, and include explicit manual cleanup commands: remove the directory, then run `git worktree prune`. | Still safe by default; gives users the complete next step; improves human and AI-agent actionability. | Longer error text; still requires manual cleanup. |
| Controlled removal fallback | On the submodule failure, delete the worktree directory and run `git worktree prune` automatically, at least when `--force` is set. | Most ergonomic when the user already requested force. | More destructive than the current `git worktree remove --force` contract; bypasses Git's refusal; needs extra path-safety and dirty-state guarantees. |

The recommended behavior is guided remediation because it matches the existing
safe-default direction: `ww clean` has no prompt, `ww remove --force` can discard
dirty work, and both commands are scriptable, so `ww` should not silently expand
Git's force semantics into a recursive directory deletion. A controlled fallback
can be reconsidered later as a separate opt-in feature if users want it, but it
should not be introduced as implicit error recovery.

No ADR update is planned for this bug fix because the decision is an error
handling policy within an existing command contract. If the later implementation
chooses automatic directory deletion, add an ADR before coding that change.

## Spec Changes

Update `docs/specs/cli-commands.md`:

- In `ww remove`, document that if `git worktree remove` fails because the
  target contains submodules, `ww` exits non-zero with an actionable diagnostic.
- Require the text diagnostic to include:
  - the target worktree path
  - a statement that Git cannot remove worktrees containing submodules
  - manual remediation commands equivalent to `rm -rf <worktree-path>` and
    `git worktree prune`
  - a warning that manual directory removal permanently deletes uncommitted work
- Require `--json` failure output to remain an error, not a success result. The
  current command-level error path can remain non-JSON unless execution decides
  to add a structured error envelope consistently across command failures.
- In `ww clean`, clarify that submodule removal failures are reported per
  worktree just like other removal failures, clean continues with later
  candidates, and the final command exits non-zero if any target fails.

Update `docs/specs/git-operations.md`:

- Extend the `git worktree remove` section with the known Git limitation for
  worktrees containing submodules.
- Document that `ww` detects that failure by matching the Git error text and
  returns guided remediation rather than retrying with recursive deletion.

## Implementation Scope

Code changes should stay focused on the shared removal path:

| File | Planned change |
| --- | --- |
| `git/git.go` | Add a helper or typed predicate for recognizing Git's submodule worktree-remove failure from command output. Keep the low-level runner Git-native. |
| `worktree/worktree.go` | In `Manager.Remove`, wrap the specific submodule failure with a clearer error that includes the path and remediation guidance. Leave normal removal errors unchanged except for preserving the original cause. |
| `cmd/ww/sub_remove.go` | No separate removal algorithm. Verify the shared error text is surfaced cleanly for text mode and does not produce misleading success JSON. |
| `cmd/ww/sub_clean.go` | Preserve current bulk behavior: report the failed target, continue later targets, and return the final aggregate failure. Ensure the submodule diagnostic is visible enough in both text and JSON `error` fields. |
| `integration_test.go` | Add host integration coverage for `ww remove --force` against a worktree with an initialized submodule. Add a `ww clean --force` regression if the setup can mark the branch cleanable without making the test brittle. |
| `git/git_test.go` or `worktree/worktree_test.go` | Add focused unit coverage for the submodule-error recognizer and `Manager.Remove` wrapping behavior without requiring a real submodule checkout. |

Do not add an automatic `rm -rf` or `git worktree prune` fallback in this plan's
execution unless the human explicitly changes the decision during review.

## Sub-tasks

- [ ] [parallel] Update `docs/specs/cli-commands.md` with the submodule removal
  error contract for `ww remove` and `ww clean`.
- [ ] [parallel] Update `docs/specs/git-operations.md` with Git's submodule
  limitation and `ww`'s detection/remediation behavior.
- [ ] [depends on: specs] Add a submodule-removal failure recognizer close to
  the Git runner or removal manager boundary.
- [ ] [depends on: recognizer] Wrap the specific failure in
  `worktree.Manager.Remove` with a guided remediation message that preserves the
  original error context.
- [ ] [depends on: removal wrapping] Verify `ww remove` text mode,
  `ww remove --json`, `ww clean` text mode, and `ww clean --json` surface the
  failure without reporting success.
- [ ] [depends on: specs, removal wrapping] Add focused unit tests for
  recognizer/wrapping behavior.
- [ ] [depends on: removal wrapping] Add host integration reproduction coverage
  for `ww remove --force` with a submodule-containing worktree.
- [ ] [depends on: integration] Add `ww clean --force` integration coverage if
  the cleanability setup remains straightforward; otherwise document why the
  shared `Manager.Remove` unit coverage is sufficient.
- [ ] [depends on: all above] Move
  `docs/issues/worktree-remove-fails-with-submodules.md` to `docs/issues/done/`
  during the implementation PR after verification passes.

## Verification

- Run `make test`.
- Run `make lint`.
- Run the new focused integration test for submodule removal; if the test suite
  requires full integration execution, use `make test-integration`.
- Manually verify a disposable repo with a populated submodule produces the
  guided remediation error for `ww remove --force`.
- Verify `ww clean` continues processing later cleanable worktrees after one
  submodule-containing worktree fails and exits non-zero at the end.

## Out of Scope

- Automatically deleting the worktree directory.
- Running `git worktree prune` automatically.
- Changing `ww remove --force` semantics beyond improving this specific error.
- Adding new CLI flags for destructive fallback cleanup.
- Implementing lifecycle hooks or broader submodule management.
