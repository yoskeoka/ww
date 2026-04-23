# Agentic Review Workflows

This spec covers the GitHub Agentic Workflow review checks:

- `.github/workflows/plan-review.md`
- `.github/workflows/impl-review.md`
- `.github/workflows/spec-code-sync.md`

Each source workflow is compiled into its corresponding `.lock.yml` file by `gh aw compile`.

## Advisory Comment Contract

The `plan-review`, `impl-review`, and `spec-code-sync` workflows provide advisory review feedback through PR comments. They must not create formal GitHub PR reviews and must not dismiss prior PR reviews.

Each workflow must define a custom safe-output job that upserts a PR issue comment. The safe-output item must include:

- `event`: either `APPROVE` or `REQUEST_CHANGES`
- `body`: markdown review feedback

## Markers

Each workflow comment must include exactly one stable hidden marker:

- `plan-review`: `<!-- gh-aw:plan-review -->`
- `impl-review`: `<!-- gh-aw:impl-review -->`
- `spec-code-sync`: `<!-- gh-aw:spec-code-sync -->`

The visible comment body must include the decision signal (`APPROVE` or `REQUEST_CHANGES`) and detailed review feedback.

## Upsert Behavior

The safe-output job must:

1. Read the agent safe-output JSON.
2. Extract all comment-upsert items for the workflow.
3. List PR issue comments with pagination.
4. Filter comments authored by `github-actions[bot]` whose body contains the workflow marker.
5. Sort matching comments by `id`.
6. Keep the newest matching comment.
7. Delete older matching duplicates on a best-effort basis.
8. Update the newest matching comment, or create a new comment if none exists.

Comment deletion failures are non-fatal and must be emitted as warnings.

The job must request `issues: write` permission so PR issue comments can be created and updated consistently.

Verification for this spec is `gh aw compile --strict` for the three covered workflows, plus a stale-token search that confirms the current workflow sources and lock files do not contain legacy formal-review output names or Pull Requests review API calls.

## Prohibited Behavior

These workflows must not:

- define or instruct agents to call the legacy formal-review safe output
- expose or rely on the built-in add-comment safe output
- call Pull Requests API methods that create formal reviews
- call Pull Requests API methods that dismiss formal reviews
- include instructions about enabling Actions-authored PR approvals

Because feedback is advisory PR-comment state, repository settings for Actions-authored PR approvals are not required.

## Agent Prompt Requirements

Each prompt must require the agent to produce exactly one review-comment safe output when it can read PR content. The comment body must include:

- the stable workflow marker
- the decision label (`APPROVE` or `REQUEST_CHANGES`)
- specific, actionable feedback

The prompt must identify the custom safe-output tool by the runtime-visible tool name `safeoutputs-upsert_pr_comment`, while also allowing the unqualified `upsert_pr_comment` name when a runtime exposes it that way. The prompt must explicitly tell agents not to use the built-in add-comment tool for this output.

The agent may call `noop` only when it is completely unable to read PR content.
