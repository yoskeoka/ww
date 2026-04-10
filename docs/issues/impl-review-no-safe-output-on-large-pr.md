# impl-review produces no safe output on large PRs

## Summary

The `impl-review` agentic workflow sometimes completes the agent job without
producing any safe output (no `submit_pr_review` and no `noop` call). This
causes a failure issue to be filed automatically.

## Observed behavior

Workflow run: https://github.com/yoskeoka/ww/actions/runs/24210343007
PR: #114 (diff ≈ 37 KB, 11 files changed)

The agent job ran for ~4.5 minutes and made 28 inference requests to
`api.githubcopilot.com`, but `agent_output.json` was never created. The
conclusion job logged:
```
Agent succeeded but produced no safe outputs
```

## Root cause

The agent analyzed the PR in its internal LLM reasoning (many inference
calls) but ran out of context space or reached a token limit before it could
call `submit_pr_review`. Because the instructions did not mention calling
`noop` as a fallback, the agent produced zero output instead.

## Fix

Updated `impl-review.md` and `plan-review.md` with:

1. An explicit note that **zero output is not acceptable** — the agent must
   call either `submit_pr_review` or `noop`.
2. A `noop` fallback path: if the agent was unable to read any PR content,
   it should call `noop` with an explanation rather than silently exiting.
3. A **Reading Strategy** section that guides the agent to read files in a
   priority order and submit the review as soon as a confident decision can
   be made, rather than trying to read every line before deciding.

## Related

- Issue #116: the automatically filed workflow failure issue that triggered this fix
- PR #114: the implementation PR where the workflow failure occurred
