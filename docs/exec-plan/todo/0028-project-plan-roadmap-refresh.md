# 0028: Project Plan Roadmap Refresh

> **Execution**: Use `/execute-task` to implement this plan.

**Objective:** Refresh `ww`'s `docs/project-plan.md` so it clearly shows which FR/NFR items are already implemented, re-prioritizes remaining work against current real-world usage, and adds the next roadmap phase around global configuration and stronger hook workflows for sandboxed agent environments.

## Context

Current `project-plan.md` still reflects older assumptions in a few places:

- FR/NFR entries do not show implementation status, making it hard to see what is already shipped versus still pending.
- `FR-15` (ship agent skill files) is no longer a near-term priority for the current primary user because `ww create`, `ww cd`, and a documented `--sandbox` example already cover the common agent workflow.
- The highest-friction current workflow gap is not agent discoverability but config/hook portability across many repositories without committing `.ww.toml` into each target repo.
- `FR-25` currently assumes upstream Claude Code sandbox issues must be resolved first. That framing needs tightening now that Issue #13195 is closed: the roadmap should distinguish between the parts `ww` can improve itself (global config, sandbox-friendly hook workflows, documented safe patterns) and the parts still limited by external sandbox behavior.

Past decisions reviewed before this change:

- `docs/design-decisions/core-beliefs.md`: keep the plan explicit and correctness-oriented; avoid speculative roadmap sprawl.
- `docs/design-decisions/adr.md`: sandbox mode is intentionally bounded to the current workspace/repo root. Any new config direction should preserve that bounded model instead of reintroducing implicit parent scanning.

## Prioritization Summary

Options considered for the next unimplemented phase:

1. Agent skill packaging first (`FR-15`)
2. Global config and hook workflow improvements first
3. Full Claude Code sandbox compatibility first (`FR-25`)

Recommendation: choose option 2.

Reasoning:

- It directly addresses current user pain: keeping hooks/config in one global place, applying them per target project, and supporting sandbox-friendly sharing patterns such as symlinking `.env` or dependency directories from the main worktree.
- It builds on already-shipped features (`copy_files`, `symlink_files`, `post_create_hook`, `--sandbox`) instead of opening a disconnected new surface.
- It should precede trust hardening. Security prompts and hook hardening are more valuable after the hook/config model is powerful enough to justify them.
- Full `FR-25` remains partly external. Closing one upstream issue does not prove that submodule-heavy `git worktree add` flows are now fully compatible across real Claude Code sandbox environments.

## Docs Changes

| File | Change |
|------|--------|
| `docs/project-plan.md` | Add explicit status markers for each FR/NFR, refresh requirement wording where the implementation diverged, add the next roadmap phase for global config + hook workflows, and reorder remaining phases by current priority |

## Spec Changes

None in this plan. This is a roadmap and requirement-status refresh only.

## Code Changes

None in this plan.

## Sub-tasks

- [ ] Add a short status legend to `docs/project-plan.md`
- [ ] Mark each FR and NFR as implemented, partial, planned, or blocked
- [ ] Correct requirement wording where the project plan currently overstates shipped behavior (for example, `FR-1`)
- [ ] Re-prioritize future roadmap items so global config + hook workflows become the next phase
- [ ] Reframe `FR-25` and milestone wording so external blockers are explicit without hiding adjacent `ww`-owned improvements

## Parallelism

No meaningful parallel work. The roadmap refresh is a single coherent document change.

## Verification

- `docs/project-plan.md` makes FR/NFR implementation status obvious at a glance
- The next unimplemented phase matches current user pain around global config and hook workflows
- Lower-priority items such as agent skill packaging are visibly deprioritized
- `FR-25` wording no longer depends on the now-stale assumption that Issue #13195 is still open
