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
