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
              const marker = '<!-- gh-aw:plan-review -->';
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

- Call the safe-output tool named `safeoutputs-upsert_pr_comment` with `event: "APPROVE"` to post/update the advisory PR comment. If your runtime exposes the same tool without the `safeoutputs-` prefix, call `upsert_pr_comment`.
- Call the safe-output tool named `safeoutputs-upsert_pr_comment` with `event: "REQUEST_CHANGES"` to post/update the advisory PR comment. If your runtime exposes the same tool without the `safeoutputs-` prefix, call `upsert_pr_comment`.
- Call `noop` **only** if you were completely unable to read any PR content (e.g., all tool calls failed). Include a brief explanation in the message.

Include your detailed feedback in the `body` field of `upsert_pr_comment`. The body MUST include the stable marker `<!-- gh-aw:plan-review -->` and a visible `APPROVE` or `REQUEST_CHANGES` decision label.
Do not call `add_comment`, and do not pass `type: "upsert_pr_comment"` to `add_comment`.

Provide specific, actionable feedback. Reference the exact sections that need improvement.

### Fallback if safe-output tools are unavailable

If safe-output tool calls fail with `Tool "<name>" does not exist` (or the same message with single quotes, `Tool '<name>' does not exist`), do **not** end without output.

Use `shell` to write `/tmp/gh-aw/agent_output.json` directly with exactly one item in the `items` array.

**Important:** the file must contain valid JSON. Do **not** paste a raw multi-line review body directly into a JSON string. Review bodies often contain newlines, quotes, and Markdown, so generate the JSON with a serializer (for example `python -` with a heredoc or `jq -n`) or otherwise ensure newlines are encoded as `\n` and quotes/backslashes are escaped.

This is only a fallback path when the safe-output tools are unavailable.

Example shape:

```json
{"items":[{"type":"upsert_pr_comment","event":"APPROVE","body":"<!-- gh-aw:plan-review -->\nAPPROVE\n\nLGTM. Plan is complete and actionable."}]}
```

If you cannot read any PR content, use `noop` instead:

```json
{"items":[{"type":"noop","message":"Unable to read PR content because required tool calls failed."}]}
```

Example using Python JSON encoding:

```bash
python - <<'PY'
import json
body = "LGTM. Plan is complete and actionable."
obj = {"items":[{"type":"upsert_pr_comment","event":"APPROVE","body":"<!-- gh-aw:plan-review -->\nAPPROVE\n\n" + body}]}
with open("/tmp/gh-aw/agent_output.json", "w", encoding="utf-8") as f:
    json.dump(obj, f)
PY
```

### Reading Strategy

To avoid running out of context on large PRs, follow this order and stop reading once you have enough information to decide whether the plan meets the review criteria (problem statement, spec changes, concrete sub-tasks, and dependencies):

1. Read only changed files under `docs/exec-plan/todo/` first.
2. If needed, read referenced issue files under `docs/issues/` for context.
3. Skim only filenames of other changed files to confirm scope is plan-only.

Submit your review as soon as you can make a confident decision — do not wait until you have read every line.
