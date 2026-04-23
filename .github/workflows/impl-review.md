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
              const marker = '<!-- gh-aw:impl-review -->';
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

Provide specific, actionable feedback referencing the plan sub-tasks and spec sections.
### Submitting Your Review

After making your decision, submit an advisory PR comment using the `safeoutputs-upsert_pr_comment` safe output. If your runtime exposes the tool without the prefix, call `upsert_pr_comment`.

- Use `event: "APPROVE"` to approve the implementation.
- Use `event: "REQUEST_CHANGES"` to request changes.
- Include your detailed feedback in `body`.
- The body must include the stable marker `<!-- gh-aw:impl-review -->` and a visible `APPROVE` or `REQUEST_CHANGES` decision label.
- Do not call `add_comment`.
