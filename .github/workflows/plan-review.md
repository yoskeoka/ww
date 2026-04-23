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
  jobs:
    upsert_pr_comment:
      description: "Upsert an advisory PR comment with Approve or Request Changes"
      runs-on: ubuntu-latest
      permissions:
        issues: write
      inputs:
        event:
          description: "Review event: APPROVE or REQUEST_CHANGES"
          required: true
          type: choice
          options: ["APPROVE", "REQUEST_CHANGES"]
        body:
          description: "Review comment body text (markdown)"
          required: true
          type: string
      steps:
        - name: Upsert PR comment
          uses: actions/github-script@v8
          with:
            script: |
              const fs = require('fs');
              const output = JSON.parse(fs.readFileSync(process.env.GH_AW_AGENT_OUTPUT, 'utf8'));
              const items = Array.isArray(output)
                ? output
                    .filter(i => i && typeof i === 'object' && i.upsert_pr_comment)
                    .map(i => i.upsert_pr_comment)
                : Array.isArray(output?.items)
                  ? output.items
                      .filter(i => i && i.type === 'upsert_pr_comment')
                      .map(i => ({ event: i.event, body: i.body }))
                  : [];
              const marker = '<!-- gh-aw:plan-review -->';
              for (const item of items) {
                try {
                  const rawBody = String(item.body ?? '').trim();
                  const bodyWithoutMarker = rawBody.split(marker).join('').trim();
                  const bodyWithDecision = bodyWithoutMarker.includes(item.event)
                    ? bodyWithoutMarker
                    : `**${item.event}**\n\n${bodyWithoutMarker}`;
                  const commentBody = `${marker}\n${bodyWithDecision}`;
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
                    core.info(`Posted ${item.event} review as PR comment`);
                  }
                } catch (err) {
                  core.setFailed(`Failed to upsert ${item.event} review comment: ${err.message}`);
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

Provide specific, actionable feedback. Reference the exact sections that need improvement.
### Submitting Your Review

After making your decision, submit an advisory PR comment using the `safeoutputs-upsert_pr_comment` safe output. If your runtime exposes the tool without the prefix, call `upsert_pr_comment`.

- Emit exactly one safe-output item when you can read PR content.
- Use `event: "APPROVE"` to approve the plan.
- Use `event: "REQUEST_CHANGES"` to request changes.
- Include your detailed feedback in `body`.
- The body must include the stable marker `<!-- gh-aw:plan-review -->` and a visible `APPROVE` or `REQUEST_CHANGES` decision label.
- Use `noop` only if you were completely unable to read any PR content.
- Do not call `add_comment`.
