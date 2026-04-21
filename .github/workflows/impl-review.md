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
        pull-requests: write
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
                      .map(i => ({
                        pull_request_number: i.pull_request_number,
                        event: i.event,
                        body: i.body,
                      }))
                  : [];
              const marker = '<!-- gh-aw:impl-review -->';
              for (const item of items) {
                try {
                  const rawPullNumber = String(item.pull_request_number ?? '').trim();
                  const hasStrictOverride = /^[1-9][0-9]*$/.test(rawPullNumber);
                  const requestedPullNumber = hasStrictOverride ? Number(rawPullNumber) : null;
                  const isPullRequestEvent = context.eventName === 'pull_request';
                  if (rawPullNumber && !hasStrictOverride) {
                    core.warning(`Ignoring invalid pull_request_number: ${JSON.stringify(rawPullNumber)}`);
                  }
                  if (isPullRequestEvent && requestedPullNumber && requestedPullNumber !== context.issue.number) {
                    core.warning(`Ignoring pull_request_number override ${requestedPullNumber} for pull_request event #${context.issue.number}`);
                  }
                  if (!isPullRequestEvent && requestedPullNumber == null) {
                    throw new Error('pull_request_number is required for non-pull_request events');
                  }
                  const pullNumber = isPullRequestEvent
                    ? context.issue.number
                    : requestedPullNumber;
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

### Submitting Your Review

After making your decision, you MUST call one of the following safe outputs — producing zero output is not acceptable:

- Call the safe-output tool named `safeoutputs-upsert_pr_comment` with `event: "APPROVE"` to post/update the advisory PR comment. If your runtime exposes the same tool without the `safeoutputs-` prefix, call `upsert_pr_comment`.
- Call the safe-output tool named `safeoutputs-upsert_pr_comment` with `event: "REQUEST_CHANGES"` to post/update the advisory PR comment. If your runtime exposes the same tool without the `safeoutputs-` prefix, call `upsert_pr_comment`.
- Call `noop` **only** if you were completely unable to read any PR content (e.g., all tool calls failed). Include a brief explanation in the message.

Include your detailed feedback in the `body` field of `upsert_pr_comment`. The body MUST include the stable marker `<!-- gh-aw:impl-review -->` and a visible `APPROVE` or `REQUEST_CHANGES` decision label.
Do not call `add_comment`, and do not pass `type: "upsert_pr_comment"` to `add_comment`.

Provide specific, actionable feedback referencing the plan sub-tasks and spec sections.

### Fallback if safe-output tools are unavailable

If safe-output tool calls fail with `Tool '<name>' does not exist`, do **not** end without output.

Use `shell` to write `/tmp/gh-aw/agent_output.json` directly with one item.

**Important:** the file must contain valid JSON. Do **not** paste a raw multi-line review body directly into a JSON string. Review bodies often contain newlines, quotes, and Markdown, so generate the JSON with a serializer (for example `python -c` or `jq -n`) or otherwise ensure newlines are encoded as `\n` and quotes/backslashes are escaped.

Example shape:

```json
{"items":[{"type":"upsert_pr_comment","event":"APPROVE","body":"<!-- gh-aw:impl-review -->\nAPPROVE\n\nLGTM. Implementation matches the plan and specs."}]}
```

If you cannot read any PR content, use `noop` instead:

```json
{"items":[{"type":"noop","message":"Unable to read PR content because required tool calls failed."}]}
```

Example using Python JSON encoding:

```bash
python - <<'PY'
import json
body = "LGTM. Implementation matches the plan and specs."
obj = {"items":[{"type":"upsert_pr_comment","event":"APPROVE","body":"<!-- gh-aw:impl-review -->\nAPPROVE\n\n" + body}]}
with open("/tmp/gh-aw/agent_output.json", "w", encoding="utf-8") as f:
    json.dump(obj, f)
PY
```

This is only a fallback path when the safe-output tools are unavailable.

### Reading Strategy

To avoid running out of context on large PRs, follow this order and stop reading once you have enough information to decide:

1. Read only the plan file and `docs/spec-code-mapping.md` first.
2. Skim the list of changed files (filenames only) to verify they match the plan's code-change table.
3. Read the spec diff sections only (not implementation code) to check spec-code parity.
4. If you still need more detail, read specific code files selectively.

Submit your review as soon as you can make a confident decision — do not wait until you have read every line.
