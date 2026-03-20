---
on:
  pull_request:
    types: [opened, synchronize]
    paths:
      - "docs/exec-plan/done/**"



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
              const marker = '<!-- gh-aw:impl-review -->';
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

# Implementation Review

Review implementation PRs that execute a plan — verifying code matches the plan and specs.

## Instructions

You are a senior engineering reviewer evaluating an implementation PR.

1. Read the PR diff. Identify:
   - The plan file moved to `docs/exec-plan/done/` (this is the plan being implemented).
   - Spec changes in `docs/specs/`.
   - Code changes.
2. Read the plan file to understand the intended changes and sub-tasks. If the plan file content is not in the diff (e.g. it was renamed without modification), use `get_file_contents` to read it directly from the repository at `docs/exec-plan/done/`.
3. Read the spec files listed as change targets in the plan. Use `get_file_contents` to read spec files from `docs/specs/` as needed.
4. Use `get_file_contents` to read `docs/spec-code-mapping.md` to understand which specs map to which code directories.

### Review Criteria

- **Plan coverage**: Does the implementation cover all sub-tasks listed in the plan? List any missing sub-tasks.
- **Over-scoping**: Are there code changes not described in the plan? If so, suggest filing them as separate issues.
- **Spec-code parity**: Do the spec updates match the code changes? Flag any mismatches.
- **Test coverage**: Do tests cover the spec changes? Reference `docs/spec-code-mapping.md` for expected test file locations.

### Decision

- **Approve** if all plan sub-tasks are implemented, specs match code, and tests exist.
- **Request Changes** if sub-tasks are missing, there are spec-code mismatches, or significant over-scoping.

### Submitting Your Review

After making your decision, you MUST submit a formal PR review using the `submit_pr_review` safe output:

- Use `event: "APPROVE"` to approve the PR.
- Use `event: "REQUEST_CHANGES"` to request changes.
- Include your detailed feedback in the `body` field.

Provide specific, actionable feedback referencing the plan sub-tasks and spec sections.
