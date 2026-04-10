---
on:
  pull_request:
    types: [opened, synchronize]
    paths:
      - "docs/exec-plan/todo/**"

permissions:
  contents: read
  pull-requests: read

tools:
  github:
    toolsets: [context, repos, pull_requests]

network: defaults

safe-outputs:
  add-comment:
    discussions: false
  jobs:
    submit_pr_review:
      description: "Submit a PR review with Approve or Request Changes"
      runs-on: ubuntu-latest
      permissions:
        pull-requests: write
        issues: write
      inputs:
        event:
          description: "Review event: APPROVE or REQUEST_CHANGES"
          required: true
          type: choice
          options: ["APPROVE", "REQUEST_CHANGES"]
        body:
          description: "Review body text (markdown)"
          required: true
          type: string
      steps:
        - name: Submit PR review
          uses: actions/github-script@v8
          with:
            script: |
              const fs = require('fs');
              const output = JSON.parse(fs.readFileSync(process.env.GH_AW_AGENT_OUTPUT, 'utf8'));
              const items = output.items.filter(i => i.type === 'submit_pr_review');
              const marker = '<!-- gh-aw:plan-review -->';
              for (const item of items) {
                try {
                  // Route A: Best-effort dismiss of prior reviews from this workflow only
                  try {
                    const botReviews = [];
                    for await (const page of github.paginate.iterator(github.rest.pulls.listReviews, {
                      owner: context.repo.owner,
                      repo: context.repo.repo,
                      pull_number: context.issue.number,
                    })) {
                      for (const r of page.data) {
                        if (r.user && r.user.login === 'github-actions[bot]' &&
                            r.state !== 'DISMISSED' && (r.body || '').includes(marker)) {
                          botReviews.push(r);
                        }
                      }
                    }
                    for (const rev of botReviews) {
                      await github.rest.pulls.dismissReview({
                        owner: context.repo.owner,
                        repo: context.repo.repo,
                        pull_number: context.issue.number,
                        review_id: rev.id,
                        message: 'Superseded by updated review',
                      });
                    }
                  } catch (dismissErr) {
                    core.warning(`Failed to dismiss prior reviews (non-fatal): ${dismissErr.message}`);
                  }
                  await github.rest.pulls.createReview({
                    owner: context.repo.owner,
                    repo: context.repo.repo,
                    pull_number: context.issue.number,
                    event: item.event,
                    body: `${item.body}\n\n${marker}`
                  });
                  core.info(`Submitted ${item.event} review`);
                } catch (err) {
                  core.warning(`Failed to submit ${item.event} review: ${err.message}. Falling back to PR comment.`);
                  const fixGuide = `> **Fix:** Go to **Settings → Actions → General → Workflow permissions** and check **"Allow GitHub Actions to create and approve pull requests"**.`;
                  const commentBody = `${marker}\n**${item.event}** (posted as comment — review submission failed)\n\n${item.body}\n\n---\n${fixGuide}`;
                  // Route B: Find bot-authored fallback comments; update newest, delete older duplicates
                  const matchingComments = [];
                  for await (const page of github.paginate.iterator(github.rest.issues.listComments, {
                    owner: context.repo.owner,
                    repo: context.repo.repo,
                    issue_number: context.issue.number,
                  })) {
                    for (const c of page.data) {
                      if (c.user && c.user.login === 'github-actions[bot]' &&
                          typeof c.body === 'string' && c.body.includes(marker)) {
                        matchingComments.push(c);
                      }
                    }
                  }
                  // Sort ascending by id; keep last (newest), delete the rest
                  matchingComments.sort((a, b) => a.id - b.id);
                  const toDelete = matchingComments.slice(0, -1);
                  const existingId = matchingComments.length > 0 ? matchingComments[matchingComments.length - 1].id : null;
                  for (const dup of toDelete) {
                    try {
                      await github.rest.issues.deleteComment({
                        owner: context.repo.owner,
                        repo: context.repo.repo,
                        comment_id: dup.id,
                      });
                    } catch (delErr) {
                      core.warning(`Failed to delete duplicate comment ${dup.id}: ${delErr.message}`);
                    }
                  }
                  if (existingId) {
                    await github.rest.issues.updateComment({
                      owner: context.repo.owner,
                      repo: context.repo.repo,
                      comment_id: existingId,
                      body: commentBody
                    });
                    core.info(`Updated existing ${item.event} review comment`);
                  } else {
                    await github.rest.issues.createComment({
                      owner: context.repo.owner,
                      repo: context.repo.repo,
                      issue_number: context.issue.number,
                      body: commentBody
                    });
                    core.info(`Posted ${item.event} review as PR comment (fallback)`);
                  }
                }
              }
---

# Plan Review

Review execution plans in `docs/exec-plan/todo/` for completeness and quality.

## Instructions

You are a senior engineering reviewer evaluating an execution plan PR.

1. Read the PR diff to find new or modified files in `docs/exec-plan/todo/`.
2. If the plan references an issue in `docs/issues/`, use `get_file_contents` to read that file for context.
3. Evaluate the plan against these criteria:

### Review Criteria

- **Problem statement**: Does the plan clearly state the problem and proposed solution?
- **Spec changes**: Are specification changes in `docs/specs/` identified?
- **Code changes**: Are code changes broken into concrete, implementable sub-tasks?
- **Dependencies**: Are dependencies between sub-tasks specified (parallel vs sequential)?
- **Scope**: Is the scope appropriate — not too broad (should be split) and not too narrow (missing related changes)?

### Decision

- **Approve** if all criteria are met or any gaps are minor.
- **Request Changes** if the plan is missing critical information (problem statement, spec changes, or sub-tasks), the scope is clearly wrong, or dependencies are missing that would cause implementation failures.

### Submitting Your Review

After making your decision, you MUST call one of the following safe outputs — producing zero output is not acceptable:

- Call `submit_pr_review` with `event: "APPROVE"` to approve the PR.
- Call `submit_pr_review` with `event: "REQUEST_CHANGES"` to request changes.
- Call `noop` **only** if you were completely unable to read any PR content (e.g., all tool calls failed). Include a brief explanation in the message.

Include your detailed feedback in the `body` field of `submit_pr_review`.

Provide specific, actionable feedback. Reference the exact sections that need improvement.
