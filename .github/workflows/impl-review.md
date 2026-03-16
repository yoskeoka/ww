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
