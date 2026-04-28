# Plan Review `upsert_pr_comment` fails with integration-permission error

## Summary

The `Plan Review` workflow can fail in the `upsert_pr_comment` job even after
the agent artifact downloads successfully. In PR #213, job
`73463829880` failed while trying to create or update the durable PR comment
from the agent's `APPROVE` output.

Observed run:

- workflow run: `25074277852`
- job: `upsert_pr_comment` (`73463829880`)
- PR: `#213`
- failing API call path: `github.rest.issues.updateComment()` or
  `github.rest.issues.createComment()` inside `actions/github-script`
- terminal error: `Failed to upsert APPROVE review comment: Resource not accessible by integration`

The job log showed `Issues: write` permission in the runner summary, so the
failure appears to be a workflow/event permission mismatch or a token-scope
constraint rather than a missing artifact or malformed agent output.

## Impact

The workflow review run reports failure even when the underlying plan-review
logic succeeded and produced a valid agent output artifact. That creates noisy
CI and makes it harder to distinguish workflow plumbing failures from actual
review findings.

## Proposed Solution

Investigate the permission model for the `Plan Review` `upsert_pr_comment`
job and confirm whether this path is running under an event/token combination
that cannot write issue comments.

Concrete checks:

- Confirm which event triggered run `25074277852` and whether that event grants
  comment-write access to the token actually used by `github-script`.
- Check whether the job is operating on a PR from a branch or context where
  comment mutation is intentionally restricted.
- Compare the `Plan Review` comment-upsert job with the corresponding
  `Implementation Review` path to see whether the failure is specific to one
  workflow or shared comment-upsert logic.
- If the token scope is correct, log the exact API endpoint that returns
  `Resource not accessible by integration` so the failing permission boundary
  is explicit in future runs.

## Priority

Medium. This does not block the underlying code verification, but it causes a
review workflow to fail for workflow-infrastructure reasons and should be fixed
before relying on the durable PR-comment path.
