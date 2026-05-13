# Remove gh-aw Agentic Workflow
**Execution**: Use `/execute-task` to implement this plan.

## Objective

Remove the current `gh aw`-based PR review workflow stack from `ww` so the repository stops carrying unreliable advisory automation and unnecessary CI failure noise.

This plan only covers removal of the current workflow line and the supporting repository documentation/spec updates. It does not decide how or whether agentic review automation should be reintroduced later.

## Context

- `ww` currently carries three `gh aw` review workflows: `plan-review`, `impl-review`, and `spec-code-sync`.
- The current line has unresolved runtime and permission issues recorded in local issue docs such as `docs/issues/gh-aw-custom-safe-output-not-emitted.md` and `docs/issues/plan-review-upsert-pr-comment-permission.md`.
- The operator decision for this plan is to remove the current workflows now rather than continue paying CI noise and maintenance cost for unreliable review automation.

Past decisions reviewed before planning:

- `docs/project-plan.md`: `ww` values fast, portable, agent-friendly workflows, but that does not require keeping CI-side agentic review when it is no longer reliable.
- `docs/design-decisions/core-beliefs.md`: correctness and explicit workflow contracts matter more than preserving an unstable automation path.
- `docs/design-decisions/adr.md`: no existing ADR requires these review workflows to remain present, so removal can be handled as a focused workflow/spec change unless execution uncovers a broader policy decision.

## Spec Changes

Update the specs and durable docs so they describe the post-removal state accurately:

- remove or retire `docs/specs/agentic-review-workflows.md`
- update `docs/spec-code-mapping.md` so it no longer maps removed `gh aw` workflow assets to the retired spec surface
- update `docs/specs/github-actions-pinning.md` so it no longer treats `gh aw` review workflow sources and generated lock files as active repo assets that must be preserved
- update `AGENTS.md` to remove the `gh-aw` automated review section and any workflow-editing guidance that becomes stale after removal
- update any remaining durable docs that still instruct contributors to maintain or regenerate these workflows

## Code and Repository Changes

Remove the workflow assets and supporting repository metadata for the current `gh aw` review setup:

- delete `.github/workflows/plan-review.md`
- delete `.github/workflows/impl-review.md`
- delete `.github/workflows/spec-code-sync.md`
- delete the generated lock files for those workflows
- delete `.github/aw/actions-lock.json` if the inventory confirms it is only retained for this `gh aw` review workflow line
- remove any now-unused `gh aw` repository assets that only exist to support these review workflows, if confirmed unused after a repo-wide sweep
- update any remaining GitHub-side workflow/config references that assume these review checks still exist

## Sub-tasks

- [ ] [parallel] Inventory every repo file that still references the `gh aw` review workflows or their generated artifacts.
- [ ] [parallel] Decide the durable documentation state after removal: delete the dedicated spec, or replace it with a short note that the repository currently has no agentic PR review workflows.
- [ ] [depends on: inventory] Update specs and durable docs to describe the removed state before touching workflow files.
- [ ] [depends on: inventory] Remove the three source workflows, their generated lock files, and any now-dead support assets that are confirmed exclusive to this workflow line.
- [ ] [depends on: spec/doc updates, workflow removal] Run repo verification for the docs/workflow diff and confirm no stale `gh aw` review references remain in active files.

## Design Decisions

- Keep the scope narrow: this plan removes the current workflow line; it does not redesign CI review automation.
- Do not auto-close the broader reintroduction discussion when this removal lands. Choosing a future backend agent and CI credential model remains separate follow-up work.
- Prefer deleting stale workflow-specific docs over leaving speculative “temporary disable” wording behind, unless execution finds a repo convention that requires an explicit historical note.

## Parallelism

- The inventory sweep and the durable-doc-state decision can happen in parallel.
- Workflow file removal depends on the inventory sweep so we do not leave behind broken references.
- Verification depends on both the doc/spec updates and the actual asset removal.

## Verification

- `make lint`
- `rg -n "gh aw|gh-aw|agentic review|plan-review|impl-review|spec-code-sync" .github docs AGENTS.md README.md`
- any repo-specific workflow/documentation checks needed after the final diff is known

## Out of Scope

- deciding which backend AI agent or API key model to use for any future CI review automation
- replacing the removed workflows with Codex-, Copilot-, or API-key-based alternatives in the same PR
- resolving the broader product/operations question of whether `ww` should have agentic PR review automation at all
