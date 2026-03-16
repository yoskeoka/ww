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
                  event: item.inputs.event,
                  body: item.inputs.body
                });
                core.info(`Submitted ${item.inputs.event} review`);
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

After making your decision, you MUST submit a formal PR review using the `submit_pr_review` safe output:

- Use `event: "APPROVE"` to approve the PR.
- Use `event: "REQUEST_CHANGES"` to request changes.
- Include your detailed feedback in the `body` field.

Provide specific, actionable feedback. Reference the exact sections that need improvement.
