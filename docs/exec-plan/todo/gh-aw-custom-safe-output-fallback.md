# gh-aw Custom Safe Output Fallback

**Execution**: Use `/execute-task` to implement this plan.

## Objective

Fix `docs/issues/gh-aw-custom-safe-output-not-emitted.md`: the agentic review workflows expose a custom `upsert_pr_comment` safe-output job, but Copilot has not reliably emitted a custom safe-output item that makes `needs.agent.outputs.output_types` contain `upsert_pr_comment`. In observed PR #190 runs, Copilot wrote fallback JSON such as `/tmp/gh-aw/agent_output.json`, but the compiled custom job stayed `SKIPPED`, so the advisory comment was not upserted.

This plan keeps the current comment-based review model from `docs/exec-plan/done/comment-based-agentic-reviews.md`: advisory workflow state belongs in marker-based PR issue comments, not formal PR reviews.

## Context

- Project goal: `ww` is agent-friendly and values predictable machine-readable automation.
- Core belief: correctness over speed; update specs before workflow code.
- Prior decision: comment-based agentic reviews replaced formal review mutation because stale formal reviews were unreliable. This fix should preserve that direction.
- gh-aw custom safe-output documentation says custom jobs receive agent data through `GH_AW_AGENT_OUTPUT`, and `items[].type` should match the custom job name with dashes converted to underscores. For `upsert_pr_comment`, the existing fallback type is therefore structurally plausible; the failure is in the path from Copilot output collection to the generated custom job trigger.

## Trade-offs

### Option A: Prompt-only tool-name tightening

Update only the three prompts to more strongly instruct Copilot to call `upsert_pr_comment` or `safeoutputs-upsert_pr_comment`.

- Pros: Smallest diff; no workflow mechanics change.
- Cons: Observed runs already contained explicit instructions and still produced fallback JSON. This does not address the case where the safe-output tool is unavailable or ignored.
- Recommendation: Do not rely on this alone.

### Option B: Make fallback output a first-class supported path

Keep direct safe-output tool invocation as the preferred path, but make fallback JSON trigger the same upsert behavior instead of being skipped. During execution, verify the current gh-aw compiler behavior and choose the smallest supported mechanism, such as:

- changing fallback output location/shape to one that `collect_ndjson_output.cjs` includes in `output_types`
- adjusting the source workflow's custom job configuration so the upsert job can run when fallback `agent_output.json` exists
- using a gh-aw-supported script/custom job route that consumes `GH_AW_AGENT_OUTPUT` without depending on a missing custom output type

- Pros: Fixes the observed failure mode directly while keeping least-privilege safe-output separation.
- Cons: Requires careful validation against generated lock files and may need gh-aw-version-specific notes.
- Recommendation: Use this approach.

### Option C: Reintroduce built-in `add-comment` as a fallback

Add the built-in safe output back so Copilot can emit comments through a known supported path.

- Pros: Likely easy for Copilot to call.
- Cons: Violates the current spec because built-in comments do not upsert marker comments and can recreate duplicate/stale advisory comments.
- Recommendation: Reject.

## Spec Changes

Update `docs/specs/agentic-review-workflows.md` before workflow changes:

- Clarify that direct custom safe-output tool calls are preferred, but fallback output must also be processed by the compiled workflow.
- Define the accepted fallback contract precisely:
  - valid JSON object with `items`
  - exactly one item for normal review completion
  - `type: "upsert_pr_comment"` for advisory comments or `type: "noop"` when PR content cannot be read
  - `event`, `body`, and optional `pull_request_number` fields for `upsert_pr_comment`
  - required `message` field for `noop` items so fallback output remains valid against the safe-output schema
- State that fallback output must not leave the custom upsert path skipped when a valid `upsert_pr_comment` item exists.
- Keep the prohibition on built-in `add-comment` and formal PR review APIs.

## Code Changes

Modify the three gh-aw source workflows and regenerate their lock files:

- `.github/workflows/plan-review.md`
- `.github/workflows/impl-review.md`
- `.github/workflows/spec-code-sync.md`
- `.github/workflows/plan-review.lock.yml`
- `.github/workflows/impl-review.lock.yml`
- `.github/workflows/spec-code-sync.lock.yml`

Expected source-workflow changes:

- Keep the existing `safe-outputs.jobs.upsert_pr_comment` job and marker-specific upsert script.
- Update prompt fallback instructions so the fallback path matches the verified gh-aw collection mechanism.
- If the verified gh-aw mechanism requires job-level configuration changes, apply the same change to all three workflows.
- Keep workflow-specific markers unchanged:
  - `<!-- gh-aw:plan-review -->`
  - `<!-- gh-aw:impl-review -->`
  - `<!-- gh-aw:spec-code-sync -->`

Do not hand-edit lock files except as generated output from `gh aw compile --strict` or the repository's documented compile command.

## Sub-tasks

- [ ] [parallel] Confirm current gh-aw compiler/runtime expectations for custom safe-output tools and fallback output paths using official docs and generated lock files.
- [ ] [parallel] Inspect recent failed workflow artifacts/logs, if accessible, to confirm whether `/tmp/gh-aw/agent_output.json`, `/tmp/gh-aw/safeoutputs.jsonl`, or MCP logs contain the valid `upsert_pr_comment` payload.
- [ ] [depends on: compiler/runtime expectations] Update `docs/specs/agentic-review-workflows.md` with the fallback processing contract.
- [ ] [depends on: spec update] Update all three workflow source files with the verified fallback mechanism and any needed prompt wording.
- [ ] [depends on: workflow source updates] Recompile all three gh-aw workflows with `gh aw compile --strict` or the repo's equivalent command.
- [ ] [depends on: lock regeneration] Move `docs/issues/gh-aw-custom-safe-output-not-emitted.md` to `docs/issues/done/gh-aw-custom-safe-output-not-emitted.md`.
- [ ] [depends on: all changes] Verify stale tokens and generated behavior:
  - lock files still expose the `upsert_pr_comment` custom safe-output job
  - valid fallback output can reach the upsert processing path
  - workflows still do not contain legacy formal review API calls
  - workflows still do not expose or instruct built-in `add-comment`

## Verification

- `gh aw compile --strict .github/workflows/plan-review.md`
- `gh aw compile --strict .github/workflows/impl-review.md`
- `gh aw compile --strict .github/workflows/spec-code-sync.md`
- `rg -n "submit_pr_review|createReview|dismissReview|add-comment|add_comment" .github/workflows docs/specs/agentic-review-workflows.md`
- `rg -n "upsert_pr_comment|agent_output.json|output_types|safeoutputs" .github/workflows/*.md .github/workflows/*.lock.yml docs/specs/agentic-review-workflows.md`

If gh-aw provides a local dry-run or fixture command for safe-output collection, add a minimal fixture that proves a fallback JSON item with `type: "upsert_pr_comment"` is counted or otherwise processed by the selected mechanism.

## Design Decisions

No new ADR is expected. This plan preserves the existing comment-based advisory review direction and tightens the implementation contract. If execution discovers that gh-aw cannot support reliable custom comment upsert without built-in `add-comment` or direct GitHub write permissions in the agent job, record that as a design decision before changing the workflow model.
