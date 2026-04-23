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
              const marker = '<!-- gh-aw:spec-code-sync -->';
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

Be precise. Reference the specific spec section and code file that are out of sync.
### Submitting Your Review

After making your decision, submit an advisory PR comment using the `safeoutputs-upsert_pr_comment` safe output. If your runtime exposes the tool without the prefix, call `upsert_pr_comment`.

- Use `event: "APPROVE"` to approve the PR.
- Use `event: "REQUEST_CHANGES"` to request changes.
- Include your detailed feedback in `body`.
- The body must include the stable marker `<!-- gh-aw:spec-code-sync -->` and a visible `APPROVE` or `REQUEST_CHANGES` decision label.
- Do not call `add_comment`.
