# Comment-Based Agentic Reviews

**Execution**: Use `/execute-task` to implement this plan.

## Objective

Change the three GitHub Agentic Workflow review checks from formal PR review submission to durable PR comments.

Today `plan-review`, `impl-review`, and `spec-code-sync` ask the agent to emit `submit_pr_review`. A custom safe-output job then tries to dismiss prior bot-authored PR reviews and create a new formal PR review. If that fails, it falls back to a marker-based PR comment. In practice, repeated pushes often leave stale reviews because the dismissal path is best-effort and does not reliably clear every previous review.

This plan makes the currently secondary comment path the primary behavior:

- each workflow posts exactly one marker-based PR comment per workflow per PR
- later runs update that comment instead of creating formal reviews
- formal PR review creation and prior-review dismissal are removed
- review state is advisory signal only, not branch-protection review state

This supports the project-plan goals that `ww` remains agent-friendly and predictable. For these workflows, predictable advisory feedback is more valuable than an unreliable formal review gate.

## Current Context

- `docs/project-plan.md`: `ww` is explicitly AI-agent friendly and the workflow/review tooling should reduce operator friction.
- `docs/design-decisions/core-beliefs.md`: correctness and spec-first workflow matter more than preserving a fragile speed/convenience mechanism.
- `docs/design-decisions/adr.md`: the existing decisions favor explicit, predictable agent/operator contracts.
- `docs/issues/done/gh-aw-pr-review-submit.md`: the original implementation introduced `submit_pr_review` because `add-comment` could not produce formal review state.
- `docs/issues/done/gh-aw-duplicate-reviews.md`: later work tried to dismiss prior reviews and keep a single fallback comment, but review dismissal is still unreliable in practice.

## Design Options

### Option A: Keep PR reviews and improve dismissal again

Keep `submit_pr_review` as the primary path and make the dismissal query broader or more defensive.

Pros:

- preserves formal `APPROVE` / `REQUEST_CHANGES` review states
- can theoretically remain usable as a branch-protection signal

Cons:

- continues depending on best-effort dismissal of historical reviews
- stale formal reviews can keep confusing the PR timeline
- every workflow must keep custom JavaScript for review listing, dismissal, creation, and fallback

### Option B: Make marker-based PR comments primary

Remove the custom `submit_pr_review` job and instruct agents to call `add_comment` directly with a stable hidden marker and a visible decision label such as `APPROVE` or `REQUEST_CHANGES`. The implementation updates or replaces the generated lock files so each workflow has a single comment output path.

Pros:

- matches the behavior the user requested
- avoids stale formal reviews and removes review dismissal entirely
- uses the built-in gh-aw `add-comment` safe output instead of custom review-submission code
- preserves advisory feedback in the PR conversation with less workflow-specific scripting

Cons:

- no longer produces a formal GitHub review state
- branch protection cannot rely on these three workflows as required PR approvals

Recommendation: choose Option B. The user explicitly requested this direction, and it better matches the current observed failure mode.

## Spec Changes

Create or update a spec for the agentic review workflows. The execution should add a focused specification file if no existing spec owns these workflows.

Expected spec contract:

- `plan-review`, `impl-review`, and `spec-code-sync` are advisory comment workflows, not formal PR review submitters.
- Each workflow must include a stable hidden marker in its posted comment:
  - `<!-- gh-aw:plan-review -->`
  - `<!-- gh-aw:impl-review -->`
  - `<!-- gh-aw:spec-code-sync -->`
- Each workflow should leave at most one current bot-authored comment with its marker per PR.
- The comment body must include the decision signal (`APPROVE` or `REQUEST_CHANGES`) and detailed feedback.
- The workflows must not attempt `pulls.createReview` or `pulls.dismissReview`.
- The workflows must not mention repository settings for allowing Actions to approve PRs, because that permission is no longer required for this behavior.
- `docs/spec-code-mapping.md` should map the new or updated spec to the `.github/workflows/plan-review.md`, `.github/workflows/impl-review.md`, `.github/workflows/spec-code-sync.md`, and corresponding `*.lock.yml` files if the current mapping does not already cover them.

## Code and Workflow Changes

### `.github/workflows/plan-review.md`

- Remove the custom `safe-outputs.jobs.submit_pr_review` job.
- Keep built-in `safe-outputs.add-comment` with `discussions: false`.
- Update instructions so the agent must call `add_comment`, not `submit_pr_review`.
- Require the marker `<!-- gh-aw:plan-review -->` in the comment body.
- Replace fallback instructions that manually write `submit_pr_review` output with fallback instructions that write exactly one `add_comment` item if the safe-output tool is unavailable.
- Keep `noop` only for the existing "unable to read PR content" case.

### `.github/workflows/impl-review.md`

- Apply the same output contract as `plan-review`.
- Require marker `<!-- gh-aw:impl-review -->`.
- Preserve implementation-review criteria and reading strategy.

### `.github/workflows/spec-code-sync.md`

- Apply the same output contract as `plan-review`.
- Require marker `<!-- gh-aw:spec-code-sync -->`.
- Preserve spec/code sync criteria and reading strategy.

### Compiled lock files

- Recompile the three gh-aw source workflows:
  - `.github/workflows/plan-review.lock.yml`
  - `.github/workflows/impl-review.lock.yml`
  - `.github/workflows/spec-code-sync.lock.yml`
- Confirm the lock files no longer contain:
  - `submit_pr_review`
  - `pulls.createReview`
  - `pulls.dismissReview`
  - "Allow GitHub Actions to create and approve pull requests"

## Sub-tasks

- [ ] Update the agentic review workflow spec and spec-code mapping before editing workflow files.
- [ ] [parallel] Update `.github/workflows/plan-review.md` to use comment output only.
- [ ] [parallel] Update `.github/workflows/impl-review.md` to use comment output only.
- [ ] [parallel] Update `.github/workflows/spec-code-sync.md` to use comment output only.
- [ ] [depends on: workflow source updates] Recompile all three gh-aw workflows and commit the regenerated lock files.
- [ ] [depends on: workflow source updates] Search for stale `submit_pr_review`, review-dismissal, and review-approval-permission wording in workflow docs, issues, specs, and lock files; update only current active docs/specs.
- [ ] [depends on: spec and workflow updates] Run workflow verification commands and inspect the generated lock files.

## Parallelism

The three workflow source files can be edited independently after the spec contract is written. Lock-file regeneration and repository-wide stale wording checks depend on those source edits.

## Verification

- Run `gh aw compile --strict` for each changed agentic workflow, or the repository's equivalent gh-aw compile command if documented.
- Run `rg -n "submit_pr_review|pulls\\.createReview|pulls\\.dismissReview|Allow GitHub Actions to create and approve pull requests" .github/workflows docs/specs docs/spec-code-mapping.md`.
- Run `make lint` if it covers workflow or docs checks in the local environment.
- Run `make test` if workflow changes or spec-code mapping updates trigger normal repository verification expectations.

## Acceptance Criteria

- The three source workflows no longer define or instruct agents to call `submit_pr_review`.
- The three compiled lock files no longer contain a `submit_pr_review` safe-output job or PR review API calls.
- Each workflow's prompt clearly instructs the agent to post a PR comment with a stable workflow marker and decision text.
- Repeated workflow runs are intended to update or replace the workflow's current PR comment, not create or dismiss formal reviews.
- Specs and mapping document the new advisory-comment contract.

## Design Decisions

No ADR update is required unless execution discovers that these workflow comments must remain usable as branch-protection gates. This plan intentionally treats them as advisory checks rather than formal review-state gates.
