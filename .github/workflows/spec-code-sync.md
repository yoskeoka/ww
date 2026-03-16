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

After making your decision, you MUST submit a formal PR review using the `submit_pr_review` safe output:

- Use `event: "APPROVE"` to approve the PR.
- Use `event: "REQUEST_CHANGES"` to request changes.
- Include your detailed feedback in the `body` field.

Be precise. Reference the specific spec section and code file that are out of sync.
