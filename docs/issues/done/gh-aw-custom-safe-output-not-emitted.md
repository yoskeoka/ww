# gh-aw custom safe-output is not emitted by Copilot agent

Status: obsoleted by `docs/exec-plan/done/remove-gh-agentic-workflow.md`. The repository removed the `gh aw` PR review workflows instead of fixing this path.

## Summary

PR #190 changed the agentic review workflows to expose a custom `upsert_pr_comment` safe-output job. During PR follow-up, the runtime registered the custom tool as `safeoutputs-upsert_pr_comment`, but Copilot still did not emit that safe-output type directly. The agent instead wrote fallback JSON files such as `/tmp/gh-aw/agent_output.json`, which the gh-aw `output_types` detector did not count as a custom job output. As a result, the compiled `upsert_pr_comment` job stayed `SKIPPED` even though the agent reached an `APPROVE` decision.

Observed workflow runs:

- PR #190 first head `8963a4e`: agent attempted built-in `add_comment` with `type: "upsert_pr_comment"` and then wrote fallback JSON.
- PR #190 later head `459e796`: agent wrote valid fallback JSON with `{"type":"upsert_pr_comment"}`, but `needs.agent.outputs.output_types` remained empty and the custom job skipped.
- PR #190 latest head `fb7b441`: built-in `add-comment` was removed from workflow configuration, but the custom job still skipped because the agent did not emit the custom safe-output type directly.

## Proposed Solution

Investigate gh-aw/Copilot custom safe-output invocation semantics and update the workflow prompts or gh-aw configuration so agents reliably call the custom output tool instead of writing fallback JSON. Confirm whether the expected tool name is `safeoutputs-upsert_pr_comment`, `safeoutputs___upsert_pr_comment`, or another runtime-specific alias, and add a minimal reproducible workflow if needed.

## Priority

High for workflow correctness. The current PR removes stale formal reviews, but the intended durable comment upsert path depends on gh-aw detecting the custom output type and running the custom job.

## Follow-up

If CI-side agentic PR review is reconsidered in this repository, revisit this issue together with [docs/issues/0036-gh-aw-reintroduction-needs-fresh-decision.md](../0036-gh-aw-reintroduction-needs-fresh-decision.md).
