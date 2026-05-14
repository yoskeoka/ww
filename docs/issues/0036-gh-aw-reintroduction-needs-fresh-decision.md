# Reintroducing agentic PR review after `gh aw` removal needs a fresh decision

**GitHub:** https://github.com/yoskeoka/ww/issues/231
**Type:** docs | **Priority:** Medium

## Problem

The repository removed the `gh aw`-based PR review workflows because they no longer justified their operational cost, but the rationale and the restart conditions were recorded on GitHub issue `#231` instead of in `docs/issues/`.

Without a local issue, the repository loses the intended durable record for:

- why the current `gh aw` workflow line was removed
- why removal was chosen over repairing the existing workflows immediately
- what must be decided before any CI-side agentic PR review is reintroduced

## Current Decision

Keep the `gh aw` review workflows removed for now.

Reasoning:

- the recent runtime behavior no longer matched the intended review contract
- the workflows created avoidable CI noise that was hard to distinguish from real regressions
- the team has already adapted to reviewing PRs without these workflows
- the billing/contract situation for running multiple CI-side agentic reviews is not a good fit right now

## Reintroduction Blockers

Do not reintroduce agentic PR review workflows until the repository makes an explicit decision on all of the following:

- which backend agent runtime should be used in CI
- whose contract or billing model will fund the CI usage
- which API key or token model is compatible with the intended GitHub Actions execution pattern

## Revisit Together If Reintroducing

If the repository decides to reintroduce CI-side agentic review, revisit the older `gh aw` issue history at the same time instead of treating the removal as the only missing step.

- [docs/issues/done/gh-aw-custom-safe-output-not-emitted.md](done/gh-aw-custom-safe-output-not-emitted.md)
- [docs/issues/done/plan-review-upsert-pr-comment-permission.md](done/plan-review-upsert-pr-comment-permission.md)

Those issues capture concrete runtime and permission failures in the removed workflow line. Any restart decision should either confirm they no longer apply under the new backend/credential model or replace them with an explicitly different design.

## References

- `docs/exec-plan/done/remove-gh-agentic-workflow.md`
- GitHub issue `#231`
