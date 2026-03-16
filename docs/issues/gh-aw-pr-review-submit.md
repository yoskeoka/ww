# gh-aw workflows cannot submit PR reviews (Approve/Request Changes)

**Source**: PR #16 review ([comment](https://github.com/yoskeoka/ww/pull/16#discussion_r2942771326))
**Type**: feature | **Priority**: Medium
**Affects**: All three gh-aw review workflows (plan-review, impl-review, spec-code-sync)

## Problem

The three review workflows describe their output as "Approve or Request Changes" PR reviews, but the configured `safe-outputs` only includes `add-comment` — which can only post a PR **comment**, not a formal PR **review** with approve/request-changes state.

This means the workflows **cannot act as merge gates** via GitHub branch protection rules, since branch protection checks for PR review states, not comments.

### Current config (all 3 workflows)

```yaml
safe-outputs:
  add-comment:
    discussions: false
```

## Investigation

### No built-in safe output for PR reviews

The [gh-aw safe-outputs reference](https://github.github.com/gh-aw/reference/safe-outputs/) lists these PR-related safe outputs:

| Safe Output | Purpose |
|---|---|
| `add-comment` | Post a comment on a PR |
| `create-pull-request-review-comment` | Line-level review comments |
| `reply-to-pull-request-review-comment` | Reply to review threads |
| `resolve-pull-request-review-thread` | Resolve threads |

**There is no `submit-pull-request-review` or equivalent.** No built-in safe output can submit an Approve/Request Changes review.

### Solution: Custom safe output jobs

gh-aw supports [custom safe output jobs](https://github.github.com/gh-aw/reference/custom-safe-outputs/) — a `jobs:` block under `safe-outputs` that runs arbitrary Actions steps with the agent's output.

The approach:

1. **Create a shared custom job** (e.g., `.github/workflows/shared/submit-pr-review.md`) that:
   - Reads the agent's output from `$GH_AW_AGENT_OUTPUT`
   - Extracts the review decision (approve/request_changes) and body
   - Calls the GitHub API via `actions/github-script@v8` to submit a PR review
2. **Import it** in all 3 workflow `.md` files via `imports:`
3. **Add `pull-requests: write`** permission to the custom job (the agent itself stays read-only)
4. **Recompile** all `.lock.yml` files with `gh aw compile --strict`

### Example custom job structure

```yaml
safe-outputs:
  jobs:
    submit-pr-review:
      description: "Submit a PR review with Approve or Request Changes"
      runs-on: ubuntu-latest
      output: "PR review submitted!"
      permissions:
        pull-requests: write
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
              for (const item of items) {
                await github.rest.pulls.createReview({
                  owner: context.repo.owner,
                  repo: context.repo.repo,
                  pull_number: context.issue.number,
                  event: item.event,
                  body: item.body
                });
                core.info(`Submitted ${item.event} review`);
              }
```

### Changes needed per workflow

For each of `plan-review.md`, `impl-review.md`, `spec-code-sync.md`:

1. Add `imports: [shared/submit-pr-review.md]` (or inline the job)
2. Remove `add-comment` from safe-outputs (or keep it alongside for comment-only fallback)
3. Update the prompt to instruct the agent to call `submit-pr-review` with `event` and `body`
4. Recompile with `gh aw compile --strict`

### Additional considerations

- The custom job needs `pull-requests: write` permission — only the job gets it, not the agent (security preserved)
- `context.issue.number` should resolve to the triggering PR number in `pull_request` events
- The `actions/github-script@v8` action SHA should be pinned in `.github/aw/actions-lock.json`
- Consider keeping `add-comment` as well, so the agent can leave detailed comments and also submit a formal review state

## Fix

Implement the custom safe output job as described above. Create as a shared import so all 3 workflows reuse the same job definition.
