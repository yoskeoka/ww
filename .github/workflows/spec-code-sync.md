---
on:
  pull_request:
    types: [opened, synchronize]
    paths:
      - "docs/specs/**"
      - "cmd/**"
      - "git/**"
      - "internal/**"
      - "worktree/**"
      - "validate/**"

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
        pull_request_number:
          description: "Optional PR number (defaults to the current PR)"
          required: false
          type: string
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
              const items = Array.isArray(output)
                ? output
                    .filter(i => i && typeof i === 'object' && i.submit_pr_review)
                    .map(i => i.submit_pr_review)
                : Array.isArray(output?.items)
                  ? output.items
                      .filter(i => i && i.type === 'submit_pr_review')
                      .map(i => ({
                        pull_request_number: i.pull_request_number,
                        event: i.event,
                        body: i.body,
                      }))
                  : [];
              const marker = '<!-- gh-aw:spec-code-sync -->';
              for (const item of items) {
                try {
                  const parsedPullNumber = Number.parseInt(String(item.pull_request_number ?? ''), 10);
                  const pullNumber = Number.isInteger(parsedPullNumber) && parsedPullNumber > 0
                    ? parsedPullNumber
                    : context.issue.number;
                  // Route A: Best-effort dismiss of prior reviews from this workflow only
                  try {
                    const botReviews = [];
                    for await (const page of github.paginate.iterator(github.rest.pulls.listReviews, {
                      owner: context.repo.owner,
                      repo: context.repo.repo,
                      pull_number: pullNumber,
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
                        pull_number: pullNumber,
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
                    pull_number: pullNumber,
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
                    issue_number: pullNumber,
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
                      issue_number: pullNumber,
                      body: commentBody
                    });
                    core.info(`Posted ${item.event} review as PR comment (fallback)`);
                  }
                }
              }

---

# Spec/Code Sync Check

Verify that spec changes and code changes stay in sync.

## Instructions

You are a reviewer checking spec-code synchronization.

1. Read the PR diff.
2. Use `get_file_contents` to read `docs/spec-code-mapping.md` to understand which specs map to which code directories and test files.
3. Classify the changes:
   - **Code changed**: Check if the corresponding spec in `docs/specs/` is also updated.
   - **Spec changed**: Check if the corresponding code and tests are also updated.

### Review Criteria

- If code in a mapped directory changed but the corresponding spec did NOT change:
  - Is the code change purely internal (refactoring, no behavior change)? → OK, no spec update needed.
  - Does the code change alter observable behavior? → Flag: spec update likely needed.
- If a spec changed but the corresponding code did NOT change:
  - Is this a spec-only clarification (no behavior change)? → OK.
  - Does the spec describe new behavior? → Flag: code update likely needed.
- If both spec and code changed: verify they describe the same behavior.

### Decision

- **Approve** if specs and code are in sync, or if changes are purely internal/clarification.
- **Request Changes** if there is a clear mismatch between spec and code behavior.

### Submitting Your Review

After making your decision, you MUST call one of the following safe outputs — producing zero output is not acceptable:

- Call `submit_pr_review` with `event: "APPROVE"` to approve the PR.
- Call `submit_pr_review` with `event: "REQUEST_CHANGES"` to request changes.
- Call `noop` **only** if you were completely unable to read any PR content (e.g., all tool calls failed). Include a brief explanation in the message.

Include your detailed feedback in the `body` field of `submit_pr_review`.

Be precise. Reference the specific spec section and code file that are out of sync.

### Reading Strategy

To avoid running out of context on large PRs, follow this order and stop reading once you have enough information to decide:

1. Read `docs/spec-code-mapping.md` and the list of changed files (filenames only) first.
2. Read the spec diff sections only (not implementation code) to check spec-code parity.
3. If you still need more detail, read specific code files selectively.

Submit your review as soon as you can make a confident decision — do not wait until you have read every line.
